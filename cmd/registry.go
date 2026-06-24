package cmd

import (
	"fmt"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newRegistryCmds builds list/create/update/delete commands for each registry.
// All registry operations are WebSocket-only (config/<name>_registry/*).
func newRegistryCmds(f *cmdutil.Factory) *cobra.Command {
	parent := &cobra.Command{
		Use:   "registry",
		Short: "Manage area/device/entity/floor/label registries",
		Long: `Manage the area, device, entity, floor, label, and category registries.
The <id> is positional on update/delete. The category registry additionally
requires --scope (automation|script|scene|...).`,
		Example: `  hass-cli registry area list
  hass-cli registry area create --data '{"name":"Garage"}'
  hass-cli registry area update garage --data '{"name":"Garage Bay"}'
  hass-cli registry category list --scope automation`,
	}
	for _, name := range []string{"area", "device", "entity", "floor", "label"} {
		parent.AddCommand(newRegistryCmd(f, name))
	}
	parent.AddCommand(newCategoryRegistryCmd(f))
	return parent
}

// newCategoryRegistryCmd handles the category registry, which differs from the
// others by requiring a --scope (e.g. automation/script/scene) on every call.
func newCategoryRegistryCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "category",
		Short: "Operate on the category registry (requires --scope)",
	}

	for _, op := range []string{"list", "create", "update", "delete"} {
		var scope, dataJSON string
		use := op
		args := cobra.NoArgs
		if op == "update" || op == "delete" {
			use = op + " <category_id>"
			args = cobra.ExactArgs(1)
		}
		c := &cobra.Command{
			Use:   use,
			Short: fmt.Sprintf("%s category registry entries", op),
			Args:  args,
			RunE: func(cmd *cobra.Command, args []string) error {
				if scope == "" {
					return fmt.Errorf("--scope is required (e.g. automation, script, scene)")
				}
				payload := map[string]any{
					"type":  "config/category_registry/" + op,
					"scope": scope,
				}
				if op == "update" || op == "delete" {
					payload["category_id"] = args[0]
				}
				var fields map[string]any
				var err error
				switch op {
				case "create":
					fields, err = requireDataField(dataJSON, "name")
				case "update":
					fields, err = requireData(dataJSON)
				default:
					fields, err = parseDataObject(dataJSON)
				}
				if err != nil {
					return err
				}
				for k, v := range fields {
					payload[k] = v
				}
				return wsCall(f, cmd, payload)
			},
		}
		c.Flags().StringVar(&scope, "scope", "", "Category scope: automation|script|scene|...")
		if op != "list" {
			c.Flags().StringVar(&dataJSON, "data", "", "Fields as a JSON object (or @file.json)")
		}
		cmd.AddCommand(c)
	}
	return cmd
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
			return renderRawCols(f, raw, fmt.Sprintf("ID=%s_id,NAME=name", name))
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
			var fields map[string]any
			var err error
			switch op {
			case "create":
				fields, err = requireDataField(dataJSON, "name")
			case "update":
				fields, err = requireData(dataJSON)
			default:
				fields, err = parseDataObject(dataJSON)
			}
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
