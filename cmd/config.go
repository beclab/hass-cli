package cmd

import (
	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newConfigCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect Home Assistant instance configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get the running instance configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.Config(cmd.Context())
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})
	return cmd
}
