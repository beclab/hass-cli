package cmd

import (
	"net/url"
	"strings"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newSystemCmd groups read-only system insight: health, repairs, logs, and the
// native data sources that audit playbooks build on.
func newSystemCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System health, repairs, logs, and history",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Integration system health info",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.SystemHealthInfo(cmd.Context())
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "repairs",
		Short: "List open repair issues (Issue Registry)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), map[string]any{"type": "repairs/list_issues"})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "errorlog",
		Short: "Print the Home Assistant error log",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			out, err := c.ErrorLog(cmd.Context())
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write([]byte(out))
			return nil
		},
	})

	var lbStart, lbEntity string
	logbookCmd := &cobra.Command{
		Use:   "logbook",
		Short: "Logbook entries (optionally from --start, filtered by --entity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			path := ""
			if lbStart != "" {
				path += "/" + lbStart
			}
			if lbEntity != "" {
				path += "?entity=" + url.QueryEscape(lbEntity)
			}
			raw, err := c.Logbook(cmd.Context(), path)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	logbookCmd.Flags().StringVar(&lbStart, "start", "", "ISO8601 start timestamp")
	logbookCmd.Flags().StringVar(&lbEntity, "entity", "", "Filter by entity_id")
	cmd.AddCommand(logbookCmd)

	var histStart, histEntities string
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "State history (from --start, filtered by --entities a,b,c)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			path := ""
			if histStart != "" {
				path += "/" + histStart
			}
			if histEntities != "" {
				path += "?filter_entity_id=" + url.QueryEscape(strings.TrimSpace(histEntities))
			}
			raw, err := c.History(cmd.Context(), path)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	historyCmd.Flags().StringVar(&histStart, "start", "", "ISO8601 start timestamp")
	historyCmd.Flags().StringVar(&histEntities, "entities", "", "Comma-separated entity_ids")
	cmd.AddCommand(historyCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "hardware",
		Short: "Board / dongle hardware info (hardware integration)",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "hardware/info"}
		}),
	})

	cmd.AddCommand(newAnalyticsCmd(f))
	cmd.AddCommand(newLabsCmd(f))

	return cmd
}

// newAnalyticsCmd reads and updates the instance's analytics opt-in
// preferences (base/diagnostics/usage/statistics).
func newAnalyticsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Show analytics opt-in preferences",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "analytics"}
		}),
	}

	var prefData string
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set analytics preferences (--data '{\"base\":true,...}')",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prefs, err := requireData(prefData)
			if err != nil {
				return err
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), map[string]any{
				"type": "analytics/preferences", "preferences": prefs,
			})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	setCmd.Flags().StringVar(&prefData, "data", "", "Preferences JSON object (or @file.json)")
	cmd.AddCommand(setCmd)

	return cmd
}

// newLabsCmd lists experimental preview features and toggles them.
func newLabsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labs",
		Short: "List experimental preview (labs) features",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "labs/list"}
		}),
	}

	var updData string
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Toggle a preview feature (--data '{\"domain\":..,\"preview_feature\":..,\"enabled\":true}')",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseDataObject(updData)
			if err != nil {
				return err
			}
			if body == nil {
				return errLabsUpdateData
			}
			body["type"] = "labs/update"
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), body)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	updateCmd.Flags().StringVar(&updData, "data", "", "Update JSON object (or @file.json)")
	cmd.AddCommand(updateCmd)

	return cmd
}
