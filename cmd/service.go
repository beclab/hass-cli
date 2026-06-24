package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bytetrade/hass-cli/internal/catalog"
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newServiceCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "List and call services",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available services (domain -> services -> fields)",
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
			return renderRaw(f, raw)
		},
	})

	var dataJSON string
	var arguments []string
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
			for _, kv := range arguments {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("--arguments must be key=value, got %q", kv)
				}
				data[k] = coerceValue(v)
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.CallService(cmd.Context(), domain, service, data)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	callCmd.Flags().StringVar(&dataJSON, "data", "", "Service data as a JSON object (or @file.json)")
	callCmd.Flags().StringArrayVar(&arguments, "arguments", nil, "key=value pairs (repeatable)")
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
