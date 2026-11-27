package cmd

import (
	"fmt"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newEnergyCmd reads and writes the Energy dashboard preferences (the energy
// sources/devices configuration) and runs its validation. WebSocket-only.
func newEnergyCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "energy",
		Short: "Energy dashboard preferences and validation",
		Example: `  hass-cli energy prefs get
  hass-cli energy prefs save --data @energy.yaml
  hass-cli energy validate`,
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
		Short: "Save energy preferences (--data: inline JSON, or @file with YAML or JSON)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if saveData == "" {
				return fmt.Errorf("--data is required: the preferences as inline JSON, or @file with YAML or JSON")
			}
			raw, err := readDataArg(saveData)
			if err != nil {
				return err
			}
			var body map[string]any
			if err := yaml.Unmarshal(raw, &body); err != nil || len(body) == 0 {
				return fmt.Errorf("--data must be a non-empty preferences object (YAML or JSON): %v", err)
			}
			body["type"] = "energy/save_prefs"
			return wsCall(f, cmd, body)
		},
	}
	saveCmd.Flags().StringVar(&saveData, "data", "", "The preferences: inline JSON, or @file with YAML or JSON")
	prefs.AddCommand(saveCmd)
	cmd.AddCommand(prefs)

	return cmd
}
