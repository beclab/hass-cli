package cmd

import (
	"fmt"
	"strings"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newStatisticsCmd exposes the recorder's long-term statistics: which
// statistics exist, their metadata, and aggregated values over a period. These
// are the data source for energy/consumption audits. WebSocket-only.
func newStatisticsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "statistics",
		Aliases: []string{"stats"},
		Short:   "Recorder long-term statistics (list, metadata, period)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Recorder status (backlog, running, db engine)",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "recorder/info"}
		}),
	})

	var statType string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List statistic ids (--type mean|sum to filter)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"type": "recorder/list_statistic_ids"}
			if statType != "" {
				body["statistic_type"] = statType
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), body)
			if err != nil {
				return err
			}
			return renderRawCols(f, raw, colsStatistics)
		},
	}
	listCmd.Flags().StringVar(&statType, "type", "", "Filter by statistic type: mean|sum")
	cmd.AddCommand(listCmd)

	var metaIDs string
	metaCmd := &cobra.Command{
		Use:   "metadata",
		Short: "Get metadata for statistics (--ids a,b,c)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"type": "recorder/get_statistics_metadata"}
			if ids := splitCSV(metaIDs); len(ids) > 0 {
				body["statistic_ids"] = ids
			}
			return wsCall(f, cmd, body)
		},
	}
	metaCmd.Flags().StringVar(&metaIDs, "ids", "", "Comma-separated statistic_ids")
	cmd.AddCommand(metaCmd)

	var pStart, pEnd, pIDs, pPeriod string
	periodCmd := &cobra.Command{
		Use:   "period",
		Short: "Aggregated statistics over a period",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if pStart == "" {
				return fmt.Errorf("--start is required (ISO8601)")
			}
			ids := splitCSV(pIDs)
			if len(ids) == 0 {
				return fmt.Errorf("--ids is required (comma-separated statistic_ids)")
			}
			body := map[string]any{
				"type":          "recorder/statistics_during_period",
				"start_time":    pStart,
				"statistic_ids": ids,
				"period":        pPeriod,
			}
			if pEnd != "" {
				body["end_time"] = pEnd
			}
			return wsCall(f, cmd, body)
		},
	}
	periodCmd.Flags().StringVar(&pStart, "start", "", "ISO8601 start time (required)")
	periodCmd.Flags().StringVar(&pEnd, "end", "", "ISO8601 end time")
	periodCmd.Flags().StringVar(&pIDs, "ids", "", "Comma-separated statistic_ids (required)")
	periodCmd.Flags().StringVar(&pPeriod, "period", "hour", "Bucket: 5minute|hour|day|week|month")
	cmd.AddCommand(periodCmd)

	return cmd
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
