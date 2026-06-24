package cmd

import (
	"fmt"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// helperTypes are the WS-managed helper entities (input_* / counter / timer /
// schedule). Creating or deleting these is a config operation, not a service
// call, so they need typed CRUD that `service call` cannot provide.
var helperTypes = []string{
	"input_boolean",
	"input_button",
	"input_number",
	"input_select",
	"input_text",
	"input_datetime",
	"counter",
	"timer",
	"schedule",
}

// newHelperCmds builds list/create/update/delete for each helper type. Every
// type follows the same WS shape: <type>/list, <type>/create + fields,
// <type>/update + <type>_id + fields, <type>/delete + <type>_id.
func newHelperCmds(f *cmdutil.Factory) *cobra.Command {
	parent := &cobra.Command{
		Use:   "helper",
		Short: "Manage helper entities (input_*, counter, timer, schedule)",
		Example: `  hass-cli helper input_boolean list
  hass-cli helper input_boolean create --data '{"name":"Guest Mode"}'
  hass-cli helper counter update visits --data '{"step":2}'`,
	}
	for _, t := range helperTypes {
		parent.AddCommand(newHelperTypeCmd(f, t))
	}
	return parent
}

func newHelperTypeCmd(f *cmdutil.Factory, typ string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   typ,
		Short: fmt.Sprintf("Operate on %s helpers", typ),
	}
	idKey := typ + "_id"

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s helpers", typ),
		Args:  cobra.NoArgs,
		RunE: wsRunCols(f, colsHelper, func([]string) map[string]any {
			return map[string]any{"type": typ + "/list"}
		}),
	})

	var createData string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: fmt.Sprintf("Create a %s helper (--data JSON, requires name)", typ),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := parseDataObject(createData)
			if err != nil {
				return err
			}
			if fields == nil {
				return fmt.Errorf("--data is required (at least {\"name\":\"...\"})")
			}
			payload := map[string]any{"type": typ + "/create"}
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
	createCmd.Flags().StringVar(&createData, "data", "", "Helper fields as a JSON object (or @file.json)")
	cmd.AddCommand(createCmd)

	var updateData string
	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: fmt.Sprintf("Update a %s helper by id (--data JSON)", typ),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := requireData(updateData)
			if err != nil {
				return err
			}
			payload := map[string]any{"type": typ + "/update", idKey: args[0]}
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
	updateCmd.Flags().StringVar(&updateData, "data", "", "Fields to change as a JSON object (or @file.json)")
	cmd.AddCommand(updateCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <id>",
		Short: fmt.Sprintf("Delete a %s helper by id", typ),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.WS(cmd.Context(), map[string]any{"type": typ + "/delete", idKey: args[0]})
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	return cmd
}
