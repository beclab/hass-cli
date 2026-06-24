package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newRegistryCmds builds list/create/update/delete commands for each registry.
// All registry operations are WebSocket-only (config/<name>_registry/*).
func newRegistryCmds(f *cmdutil.Factory) *cobra.Command {
	parent := &cobra.Command{
		Use:   "registry",
		Short: "Manage area/device/entity/floor/label registries",
	}
	for _, name := range []string{"area", "device", "entity", "floor", "label"} {
		parent.AddCommand(newRegistryCmd(f, name))
	}
	return parent
}

func newRegistryCmd(f *cmdutil.Factory, name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Operate on the %s registry", name),
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s registry entries", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.ListRegistry(cmd.Context(), name)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	for _, op := range []string{"create", "update", "delete"} {
		cmd.AddCommand(newRegistryOpCmd(f, name, op))
	}
	return cmd
}

func newRegistryOpCmd(f *cmdutil.Factory, name, op string) *cobra.Command {
	var dataJSON string
	c := &cobra.Command{
		Use:   op,
		Short: fmt.Sprintf("%s a %s registry entry (--data JSON)", op, name),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := map[string]any{"type": fmt.Sprintf("config/%s_registry/%s", name, op)}
			if dataJSON != "" {
				var fields map[string]any
				if err := json.Unmarshal([]byte(dataJSON), &fields); err != nil {
					return fmt.Errorf("invalid --data JSON: %w", err)
				}
				for k, v := range fields {
					payload[k] = v
				}
			}
			cl, err := f.Client()
			if err != nil {
				return err
			}
			defer cl.Close()
			raw, err := cl.WS(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
	c.Flags().StringVar(&dataJSON, "data", "", "Operation fields as a JSON object")
	return c
}
