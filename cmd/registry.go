package cmd

import (
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
	idKey := name + "_id"
	// create takes only --data; update/delete require the entry id positionally.
	use := op
	args := cobra.NoArgs
	if op != "create" {
		use = op + " <id>"
		args = cobra.ExactArgs(1)
	}
	c := &cobra.Command{
		Use:   use,
		Short: fmt.Sprintf("%s a %s registry entry", op, name),
		Args:  args,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := map[string]any{"type": fmt.Sprintf("config/%s_registry/%s", name, op)}
			if op != "create" {
				payload[idKey] = args[0]
			}
			fields, err := parseDataObject(dataJSON)
			if err != nil {
				return err
			}
			for k, v := range fields {
				payload[k] = v
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
	c.Flags().StringVar(&dataJSON, "data", "", "Operation fields as a JSON object (or @file.json)")
	return c
}
