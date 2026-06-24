package cmd

import (
	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newPingCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check connectivity and authentication against the API",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.Client()
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.REST(cmd.Context(), "GET", "", nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	}
}
