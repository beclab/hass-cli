package cmd

import (
	"fmt"
	"os"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newTemplateCmd(f *cmdutil.Factory) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "template [template-string]",
		Short: "Render a Jinja template server-side",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var tmpl string
			switch {
			case file != "":
				data, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				tmpl = string(data)
			case len(args) == 1:
				tmpl = args[0]
			default:
				return fmt.Errorf("provide a template string or --file")
			}
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			out, err := c.RenderTemplate(cmd.Context(), tmpl)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Read template from a file")
	return cmd
}
