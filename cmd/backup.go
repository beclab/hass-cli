package cmd

import (
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newBackupCmd manages native Home Assistant backups over the WS backup API.
func newBackupCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "List, create, inspect, delete, and restore backups",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List backups (and backup manager state)",
		Args:  cobra.NoArgs,
		RunE:  wsRun(f, func([]string) map[string]any { return map[string]any{"type": "backup/info"} }),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "agents",
		Short: "List configured backup agents (storage locations)",
		Args:  cobra.NoArgs,
		RunE:  wsRun(f, func([]string) map[string]any { return map[string]any{"type": "backup/agents/info"} }),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <backup_id>",
		Short: "Show details for one backup",
		Args:  cobra.ExactArgs(1),
		RunE: wsRun(f, func(a []string) map[string]any {
			return map[string]any{"type": "backup/details", "backup_id": a[0]}
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <backup_id>",
		Short: "Delete a backup",
		Args:  cobra.ExactArgs(1),
		RunE: wsRun(f, func(a []string) map[string]any {
			return map[string]any{"type": "backup/delete", "backup_id": a[0]}
		}),
	})

	var createData string
	var auto bool
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup (--auto for automatic settings, else --data with agent_ids)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			var payload map[string]any
			if auto {
				payload = map[string]any{"type": "backup/generate_with_automatic_settings"}
			} else {
				fields, err := parseDataObject(createData)
				if err != nil {
					return err
				}
				if fields == nil {
					return errAgentIDsRequired
				}
				payload = map[string]any{"type": "backup/generate"}
				for k, v := range fields {
					payload[k] = v
				}
			}
			raw, err := c.WS(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	createCmd.Flags().StringVar(&createData, "data", "", `Backup params as JSON (e.g. {"agent_ids":["backup.local"],"name":"manual"}) or @file.json`)
	createCmd.Flags().BoolVar(&auto, "auto", false, "Use the configured automatic backup settings")
	cmd.AddCommand(createCmd)

	var restoreData string
	restoreCmd := &cobra.Command{
		Use:   "restore <backup_id>",
		Short: "Restore a backup (--data must include agent_id)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := parseDataObject(restoreData)
			if err != nil {
				return err
			}
			payload := map[string]any{"type": "backup/restore", "backup_id": args[0]}
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
	restoreCmd.Flags().StringVar(&restoreData, "data", "", `Restore params as JSON (e.g. {"agent_id":"backup.local"}) or @file.json`)
	cmd.AddCommand(restoreCmd)

	return cmd
}
