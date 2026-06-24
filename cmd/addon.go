package cmd

import (
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newAddonCmd manages Home Assistant add-ons through Core's supervisor/api
// proxy. Add-ons exist only on Supervised/HA OS installs; every subcommand is
// gated by requireSupervisor (via supervisorRun) so a Core/Container install
// fails with a clear message instead of an opaque Supervisor error.
func newAddonCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Manage add-ons (HA OS/Supervised installs only)",
		Example: `  hass-cli addon list
  hass-cli addon info core_mosquitto
  hass-cli addon restart core_mosquitto`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed add-ons",
		Args:  cobra.NoArgs,
		RunE: supervisorRun(f, "get", func([]string) string {
			return "/addons"
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "info <slug>",
		Short: "Show an add-on's details",
		Args:  cobra.ExactArgs(1),
		RunE: supervisorRun(f, "get", func(args []string) string {
			return "/addons/" + args[0] + "/info"
		}),
	})

	for _, verb := range []string{"start", "stop", "restart"} {
		verb := verb
		cmd.AddCommand(&cobra.Command{
			Use:   verb + " <slug>",
			Short: verb + " an add-on",
			Args:  cobra.ExactArgs(1),
			RunE: supervisorRun(f, "post", func(args []string) string {
				return "/addons/" + args[0] + "/" + verb
			}),
		})
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "logs <slug>",
		Short: "Print an add-on's recent logs",
		Args:  cobra.ExactArgs(1),
		RunE: supervisorRun(f, "get", func(args []string) string {
			return "/addons/" + args[0] + "/logs"
		}),
	})

	return cmd
}
