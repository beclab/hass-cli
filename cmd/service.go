package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/beclab/hass-cli/internal/catalog"
	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newServiceCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "List and call services",
		Example: `  hass-cli service list
  hass-cli service describe light.turn_on
  hass-cli service call light.turn_on --arguments entity_id=light.kitchen
  hass-cli service call light.turn_off --entity light.kitchen`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list [domain...]",
		Short: "List available services (domain -> services -> fields), optionally only some domains",
		Example: `  hass-cli service list light
  hass-cli service list light switch -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.Services(cmd.Context())
			if err != nil {
				return err
			}
			if len(args) > 0 {
				if raw, err = filterServiceDomains(raw, args); err != nil {
					return err
				}
			}
			if f.Output == "" || f.Output == "table" {
				return renderValueCols(f, serviceRows(raw), "SERVICE=service,DESCRIPTION=description")
			}
			return renderRaw(f, raw)
		},
	})

	var dataJSON string
	var arguments, entities []string
	callCmd := &cobra.Command{
		Use:   "call <domain.service>",
		Short: "Call a service, e.g. light.turn_on --arguments entity_id=light.kitchen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain, service, ok := strings.Cut(args[0], ".")
			if !ok {
				return fmt.Errorf("service must be <domain>.<service>, got %q", args[0])
			}
			data, err := parseDataObject(dataJSON)
			if err != nil {
				return err
			}
			if data == nil {
				data = map[string]any{}
			}
			for _, arg := range arguments {
				for _, kv := range splitArguments(arg) {
					k, v, ok := strings.Cut(kv, "=")
					if !ok {
						return fmt.Errorf("--arguments must be key=value, got %q", kv)
					}
					data[strings.TrimSpace(k)] = coerceValue(strings.TrimSpace(v))
				}
			}
			if len(entities) > 0 {
				data["entity_id"] = strings.Join(entities, ",")
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			if err := checkEntities(cmd.Context(), c, entityIDsIn(data)); err != nil {
				return err
			}
			if err := checkTarget(cmd.Context(), c, domain, service, data); err != nil {
				return err
			}
			raw, err := c.CallService(cmd.Context(), domain, service, data)
			if err != nil {
				return explainCallError(err, args[0])
			}
			return renderCallResult(f, raw)
		},
	}
	callCmd.Flags().StringVar(&dataJSON, "data", "", "Service data as a JSON object (or @file.json)")
	callCmd.Flags().StringArrayVar(&arguments, "arguments", nil, "key=value pairs (repeatable; one flag may also hold several, comma-separated). Values may be numbers, true/false or JSON lists")
	callCmd.Flags().StringSliceVar(&entities, "entity", nil, "Target entity id(s), repeatable or comma-separated; the same as --arguments entity_id=...")
	// --entity_id and --entity-id are what an agent guesses first; take them for --entity.
	callCmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "entity_id", "entity-id", "entities", "entity_ids", "entity-ids":
			name = "entity"
		}
		return pflag.NormalizedName(name)
	})
	callCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fmt.Errorf("%w\nservice call takes the target and data as: --entity light.kitchen, --arguments key=value, or --data '{\"entity_id\":\"light.kitchen\",...}'", err)
	})
	cmd.AddCommand(callCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "describe <domain.service>",
		Short: "Show a service's fields from the live service catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain, service, ok := strings.Cut(args[0], ".")
			if !ok {
				return fmt.Errorf("service must be <domain>.<service>, got %q", args[0])
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.Services(cmd.Context())
			if err != nil {
				return err
			}
			cat, err := catalog.Parse(raw)
			if err != nil {
				return err
			}
			svc, ok := cat.Lookup(domain, service)
			if !ok {
				return fmt.Errorf("service %q not found", args[0])
			}
			if f.Output == "" || f.Output == "table" {
				// A table needs rows; one object rendered as a table came out empty.
				return renderValueCols(f, fieldRows(svc), "FIELD=field,REQUIRED=required,DESCRIPTION=description,EXAMPLE=example")
			}
			return renderValue(f, svc)
		},
	})

	return cmd
}

// coerceValue turns a CLI string into a bool/number/json where it parses
// cleanly, otherwise keeps it as a string.
func coerceValue(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	var num json.Number
	if err := json.Unmarshal([]byte(s), &num); err == nil {
		if i, err := num.Int64(); err == nil {
			return i
		}
		if fl, err := num.Float64(); err == nil {
			return fl
		}
	}
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") {
		var v any
		if json.Unmarshal([]byte(s), &v) == nil {
			return v
		}
	}
	return s
}

// renderCallResult prints what a service call changed. Home Assistant answers with
// the states the call changed, so this is the confirmation and no follow-up read is
// needed. A table shows just entity, new state and name; an empty answer means
// nothing changed, usually because the target was already in that state.
// Non-table formats keep Home Assistant's body as it is.
func renderCallResult(f *cmdutil.Factory, raw json.RawMessage) error {
	if f.Output != "" && f.Output != "table" {
		return renderRaw(f, raw)
	}
	var changed []any
	if json.Unmarshal(raw, &changed) == nil && len(changed) == 0 {
		r, err := f.Renderer()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(r.Out, "Done. No entity changed state (it may already have been in the requested state).")
		return err
	}
	return renderRawCols(f, raw, colsState)
}

// fieldRows lists a service's fields one per row, sorted, for table output. A
// section's fields are listed in its place, so every name shown can be sent.
func fieldRows(svc catalog.Service) []map[string]any {
	rows := []map[string]any{}
	var add func(fields map[string]catalog.Field)
	add = func(fields map[string]catalog.Field) {
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			field := fields[name]
			if len(field.Fields) > 0 {
				add(field.Fields)
				continue
			}
			example := ""
			if len(field.Example) > 0 {
				example = string(field.Example)
			}
			rows = append(rows, map[string]any{"field": name, "required": field.Required, "description": field.Description, "example": example})
		}
	}
	add(svc.Fields)
	return rows
}

// filterServiceDomains keeps only the named domains of a /api/services answer.
func filterServiceDomains(raw json.RawMessage, domains []string) (json.RawMessage, error) {
	var all []map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return raw, nil
	}
	want := map[string]bool{}
	for _, d := range domains {
		want[strings.TrimSpace(d)] = true
	}
	kept := []map[string]any{}
	for _, d := range all {
		if name, _ := d["domain"].(string); want[name] {
			kept = append(kept, d)
		}
	}
	return json.Marshal(kept)
}

// serviceRows flattens domains into one domain.service row each, for table output.
func serviceRows(raw json.RawMessage) []map[string]any {
	var all []struct {
		Domain   string                    `json:"domain"`
		Services map[string]map[string]any `json:"services"`
	}
	_ = json.Unmarshal(raw, &all)
	rows := []map[string]any{}
	for _, d := range all {
		names := make([]string, 0, len(d.Services))
		for name := range d.Services {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			desc, _ := d.Services[name]["description"].(string)
			rows = append(rows, map[string]any{"service": d.Domain + "." + name, "description": desc})
		}
	}
	return rows
}
