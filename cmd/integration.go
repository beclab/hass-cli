package cmd

import (
	"fmt"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newIntegrationCmd manages config entries — the runtime instances of
// integrations. List/get/update/enable/disable go over WS; reload/delete are
// REST-only config endpoints.
func newIntegrationCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "integration",
		Aliases: []string{"config-entry"},
		Short:   "Manage integrations (config entries): list, reload, enable/disable, delete",
	}

	var domain string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List config entries (optionally --domain)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			payload := map[string]any{"type": "config_entries/get"}
			if domain != "" {
				payload["domain"] = domain
			}
			raw, err := c.WS(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return renderRawCols(f, raw, colsIntegration)
		},
	}
	listCmd.Flags().StringVar(&domain, "domain", "", "Filter by integration domain")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get <entry_id>",
		Short: "Get a single config entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), map[string]any{"type": "config_entries/get_single", "entry_id": args[0]})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "reload <entry_id>",
		Short: "Reload a config entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "POST", fmt.Sprintf("config/config_entries/entry/%s/reload", args[0]), nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(newConfigEntryToggleCmd(f, "enable", nil))
	cmd.AddCommand(newConfigEntryToggleCmd(f, "disable", "user"))

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <entry_id>",
		Short: "Delete a config entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "DELETE", "config/config_entries/entry/"+args[0], nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	var updateData string
	updateCmd := &cobra.Command{
		Use:   "update <entry_id>",
		Short: "Update a config entry (--data: title/pref_disable_new_entities/pref_disable_polling)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := requireData(updateData)
			if err != nil {
				return err
			}
			payload := map[string]any{"type": "config_entries/update", "entry_id": args[0]}
			for k, v := range fields {
				payload[k] = v
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	updateCmd.Flags().StringVar(&updateData, "data", "", "Fields to change as a JSON object (or @file.json)")
	cmd.AddCommand(updateCmd)

	cmd.AddCommand(newConfigFlowCmd(f))

	return cmd
}

// newConfigFlowCmd drives the data-entry config flow: discover handlers, start
// a flow for a domain, submit each step's input, and abort. This is how an
// integration is added. Discovered devices also surface here as in-progress
// flows. All flow endpoints are REST except the progress listing (WS).
func newConfigFlowCmd(f *cmdutil.Factory) *cobra.Command {
	flow := &cobra.Command{
		Use:   "flow",
		Short: "Drive config flows: add integrations, list discoveries",
	}

	var handlerType string
	handlersCmd := &cobra.Command{
		Use:   "handlers",
		Short: "List integrations that can be set up (--type integration|helper|...)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			path := "config/config_entries/flow_handlers"
			if handlerType != "" {
				path += "?type=" + handlerType
			}
			raw, err := c.REST(cmd.Context(), "GET", path, nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	handlersCmd.Flags().StringVar(&handlerType, "type", "", "Filter: integration|helper|hub|device|service")
	flow.AddCommand(handlersCmd)

	flow.AddCommand(&cobra.Command{
		Use:   "progress",
		Short: "List in-progress flows (includes discovered devices)",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "config_entries/flow/progress"}
		}),
	})

	var startData string
	startCmd := &cobra.Command{
		Use:   "start <domain>",
		Short: "Start a config flow for an integration domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"handler": args[0], "show_advanced_options": false}
			fields, err := parseDataObject(startData)
			if err != nil {
				return err
			}
			for k, v := range fields {
				body[k] = v
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "POST", "config/config_entries/flow", body)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	startCmd.Flags().StringVar(&startData, "data", "", "Extra start fields as JSON (or @file.json)")
	flow.AddCommand(startCmd)

	flow.AddCommand(&cobra.Command{
		Use:   "get <flow_id>",
		Short: "Get the current step of a flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "GET", "config/config_entries/flow/"+args[0], nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	var stepData string
	stepCmd := &cobra.Command{
		Use:   "step <flow_id>",
		Short: "Submit input for the current step (--data JSON)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseDataObject(stepData)
			if err != nil {
				return err
			}
			if body == nil {
				body = map[string]any{}
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "POST", "config/config_entries/flow/"+args[0], body)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	stepCmd.Flags().StringVar(&stepData, "data", "", "Step input as a JSON object (or @file.json)")
	flow.AddCommand(stepCmd)

	flow.AddCommand(&cobra.Command{
		Use:   "abort <flow_id>",
		Short: "Abort/delete an in-progress flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "DELETE", "config/config_entries/flow/"+args[0], nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	var ignoreTitle string
	ignoreCmd := &cobra.Command{
		Use:   "ignore <flow_id>",
		Short: "Ignore a discovered flow so it stops being suggested",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), map[string]any{
				"type": "config_entries/ignore_flow", "flow_id": args[0], "title": ignoreTitle,
			})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	ignoreCmd.Flags().StringVar(&ignoreTitle, "title", "Ignored", "Title to record for the ignored flow")
	flow.AddCommand(ignoreCmd)

	return flow
}

// newConfigEntryToggleCmd builds enable/disable, which share the
// config_entries/disable command differing only by the disabled_by value.
func newConfigEntryToggleCmd(f *cmdutil.Factory, verb string, disabledBy any) *cobra.Command {
	return &cobra.Command{
		Use:   verb + " <entry_id>",
		Short: fmt.Sprintf("%s a config entry", verb),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), map[string]any{
				"type":        "config_entries/disable",
				"entry_id":    args[0],
				"disabled_by": disabledBy,
			})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
}
