package cmd

import (
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// newAddonCmd manages Home Assistant add-ons through Core's supervisor/api
// proxy. Add-ons exist only on Supervised/HA OS installs; every subcommand is
// gated by requireSupervisor so a Core/Container install fails with a clear
// message instead of an opaque Supervisor error.
func newAddonCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Manage add-ons (HA OS/Supervised installs only)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed add-ons",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireSupervisor(cmd, f)
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.SupervisorAPI(cmd.Context(), "get", "/addons", nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "info <slug>",
		Short: "Show an add-on's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireSupervisor(cmd, f)
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.SupervisorAPI(cmd.Context(), "get", "/addons/"+args[0]+"/info", nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	for _, verb := range []string{"start", "stop", "restart"} {
		verb := verb
		cmd.AddCommand(&cobra.Command{
			Use:   verb + " <slug>",
			Short: verb + " an add-on",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				c, err := requireSupervisor(cmd, f)
				if err != nil {
					return err
				}
				defer c.Close()
				raw, err := c.SupervisorAPI(cmd.Context(), "post", "/addons/"+args[0]+"/"+verb, nil)
				if err != nil {
					return err
				}
				return renderRaw(f, raw)
			},
		})
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "logs <slug>",
		Short: "Print an add-on's recent logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireSupervisor(cmd, f)
			if err != nil {
				return err
			}
			defer c.Close()
			raw, err := c.SupervisorAPI(cmd.Context(), "get", "/addons/"+args[0]+"/logs", nil)
			if err != nil {
				return err
			}
			return renderRaw(f, raw)
		},
	})

	return cmd
}
