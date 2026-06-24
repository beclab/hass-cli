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
			raw, err := c.WS(cmd.Context(), map[string]any{"type": "system_health/info"})
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

	return cmd
}
