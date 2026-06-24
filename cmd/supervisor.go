package cmd

import (
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newSupervisorCmd exposes Supervisor-level info via Core's supervisor/api
// proxy. Only available on Supervised/HA OS installs; on Core/Container it
// fails fast with a clear message (see requireSupervisor).
func newSupervisorCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "supervisor",
		Short: "Supervisor info (HA OS/Supervised installs only)",
	}

	for _, sub := range []struct {
		use, short, endpoint string
	}{
		{"info", "Supervisor info", "/supervisor/info"},
		{"stats", "Supervisor resource stats", "/supervisor/stats"},
		{"host", "Host system info", "/host/info"},
		{"os", "Operating system info", "/os/info"},
		{"core", "Home Assistant Core info", "/core/info"},
	} {
		endpoint := sub.endpoint
		cmd.AddCommand(&cobra.Command{
			Use:   sub.use,
			Short: sub.short,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				c, err := requireSupervisor(cmd, f)
				if err != nil {
					return err
				}
				defer c.Close()
				raw, err := c.SupervisorAPI(cmd.Context(), "get", endpoint, nil)
				if err != nil {
					return err
				}
				return renderRaw(f, raw)
			},
		})
	}

	return cmd
}
