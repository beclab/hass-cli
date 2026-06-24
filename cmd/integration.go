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
			return renderRaw(f, raw)
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
			fields, err := parseDataObject(updateData)
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

	return cmd
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
