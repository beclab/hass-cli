package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newRawCmds exposes the underlying transports directly for full coverage of
// any endpoint or command not yet wrapped by a typed subcommand.
func newRawCmds(f *cmdutil.Factory) *cobra.Command {
	parent := &cobra.Command{
		Use:   "raw",
		Short: "Call the REST or WebSocket API directly (advanced)",
		Example: `  hass-cli raw api GET states/sun.sun
  hass-cli raw ws get_config
  hass-cli raw ws supervisor/api --data '{"endpoint":"/addons","method":"get"}'`,
	}

	var apiData string
	apiCmd := &cobra.Command{
		Use:   "api <method> <path>",
		Short: "Raw REST call, e.g. raw api GET states/sun.sun",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			if apiData != "" {
				raw, err := readDataArg(apiData)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return fmt.Errorf("invalid --data JSON: %w", err)
				}
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), strings.ToUpper(args[0]), args[1], body)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	apiCmd.Flags().StringVar(&apiData, "data", "", "Request body as JSON (or @file.json)")
	parent.AddCommand(apiCmd)

	var wsData string
	wsCmd := &cobra.Command{
		Use:   "ws <type>",
		Short: "Raw WebSocket command, e.g. raw ws get_config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := map[string]any{"type": args[0]}
			fields, err := parseDataObject(wsData)
			if err != nil {
				return err
			}
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
	wsCmd.Flags().StringVar(&wsData, "data", "", "Extra command fields as a JSON object (or @file.json)")
	parent.AddCommand(wsCmd)

	return parent
}
