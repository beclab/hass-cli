package cmd

import (
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newStateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Read and write entity states",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all entity states",
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
			return renderRaw(f, raw)
		},
	})

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
	setCmd.Flags().StringVar(&attrsJSON, "attributes", "", "Attributes as JSON object")
	_ = setCmd.MarkFlagRequired("state")
	cmd.AddCommand(setCmd)

	return cmd
}
