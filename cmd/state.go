package cmd

import (
	"encoding/json"
	"strings"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newStateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Read and write entity states",
		Example: `  hass-cli state list
  hass-cli state get sun.sun
  hass-cli state set sensor.foo --state 42 --attributes '{"unit_of_measurement":"C"}'`,
	}

	var domains []string
	var search string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List entity states, optionally narrowed to some domains or a name",
		Example: `  hass-cli state list --domain light
  hass-cli state list --search kitchen
  hass-cli state list --domain light,switch --search 客厅`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.States(cmd.Context())
			if err != nil {
				return err
			}
			if raw, err = filterStates(raw, domains, search); err != nil {
				return err
			}
			return renderRawCols(f, raw, colsState)
		},
	}
	listCmd.Flags().StringSliceVar(&domains, "domain", nil, "Only entities in these domains, e.g. light,switch (repeatable)")
	listCmd.Flags().StringVar(&search, "search", "", "Only entities whose entity_id or friendly name contains every word of this text (case-insensitive)")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get <entity_id>",
		Short: "Get a single entity state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.State(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	var stateValue, attrsJSON string
	setCmd := &cobra.Command{
		Use:   "set <entity_id>",
		Short: "Overwrite an entity's state object (does not drive a device)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			body := map[string]any{"state": stateValue}
			attrs, err := parseDataObject(attrsJSON)
			if err != nil {
				return err
			}
			if attrs != nil {
				body["attributes"] = attrs
			}
			raw, err := c.SetState(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	setCmd.Flags().StringVar(&stateValue, "state", "", "New state value (required)")
	setCmd.Flags().StringVar(&attrsJSON, "attributes", "", "Attributes as a JSON object (or @file.json)")
	_ = setCmd.MarkFlagRequired("state")
	cmd.AddCommand(setCmd)

	return cmd
}

// filterStates narrows a /api/states body to the requested domains and name, so an
// agent looking for one device reads a few rows instead of the whole house.
func filterStates(raw json.RawMessage, domains []string, search string) (json.RawMessage, error) {
	words := strings.Fields(strings.ToLower(search))
	if len(domains) == 0 && len(words) == 0 {
		return raw, nil
	}
	var states []map[string]any
	if err := json.Unmarshal(raw, &states); err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, d := range domains {
		if d = strings.TrimSpace(strings.TrimSuffix(d, ".")); d != "" {
			wanted[d] = true
		}
	}
	kept := []map[string]any{}
	for _, st := range states {
		id, _ := st["entity_id"].(string)
		domain, _, _ := strings.Cut(id, ".")
		if len(wanted) > 0 && !wanted[domain] {
			continue
		}
		name := ""
		if attrs, ok := st["attributes"].(map[string]any); ok {
			name, _ = attrs["friendly_name"].(string)
		}
		haystack := strings.ToLower(id + " " + name)
		match := true
		for _, w := range words {
			if !strings.Contains(haystack, w) {
				match = false
				break
			}
		}
		if match {
			kept = append(kept, st)
		}
	}
	return json.Marshal(kept)
}
