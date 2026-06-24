package cmd

import (
	"fmt"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newEnergyCmd reads and writes the Energy dashboard preferences (the energy
// sources/devices configuration) and runs its validation. WebSocket-only.
func newEnergyCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "energy",
		Short: "Energy dashboard preferences and validation",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Energy integration info (cost sensors, etc.)",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "energy/info"}
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate the current energy preferences",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "energy/validate"}
		}),
	})

	prefs := &cobra.Command{
		Use:   "prefs",
		Short: "Get or save energy preferences",
	}
	prefs.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get energy preferences (sources, device consumption)",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "energy/get_prefs"}
		}),
	})

	var saveData string
	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save energy preferences (--data with energy_sources/device_consumption)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseDataObject(saveData)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf(`--data is required (e.g. {"energy_sources":[...],"device_consumption":[...]})`)
			}
			body["type"] = "energy/save_prefs"
			return wsCall(f, cmd, body)
		},
	}
	saveCmd.Flags().StringVar(&saveData, "data", "", "Preferences as JSON (or @file.json)")
	prefs.AddCommand(saveCmd)
	cmd.AddCommand(prefs)

	return cmd
}
