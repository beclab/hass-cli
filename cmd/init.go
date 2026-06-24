package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
)

func newInitCmd(_ *cmdutil.Factory) *cobra.Command {
	o := &loginOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive first-run setup: save a server + token as a profile",
		Long: `Set up hass-cli interactively. Prompts for a Home Assistant URL and a
long-lived access token, validates them against the instance, and saves them as
a profile with the token in the OS keychain.

Flags let you skip prompts for scripted setup; pass --token-stdin to pipe the
token. On a non-interactive shell with no flags, init prints manual setup
instructions instead of prompting.`,
		Example: `  hass-cli init
  hass-cli init --name home --server http://homeassistant.local:8123 --token-stdin`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			interactive := isTTY()
			if !interactive && o.server == "" {
				printManualSetup(cmd)
				return nil
			}

			if o.name == "" && interactive {
				name, err := promptLine("Profile name [default]: ")
				if err != nil {
					return err
				}
				if name != "" {
					o.name = name
				}
			}
			o.interactive = interactive

			entry, err := runProfileLogin(cmd.Context(), o)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "\nSetup complete. Next steps:")
			fmt.Fprintln(out, "  hass-cli ping              # verify connectivity")
			fmt.Fprintln(out, "  hass-cli state list        # list entity states")
			fmt.Fprintln(out, "  hass-cli skill list        # browse bundled agent skills")
			if entry != nil {
				fmt.Fprintf(out, "\nUsing profile %q. Switch later with `hass-cli profile use <name>`.\n", entry.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&o.name, "name", "", "Profile name to create (default \"default\")")
	cmd.Flags().StringVarP(&o.server, "server", "s", "", "Home Assistant URL")
	cmd.Flags().StringVar(&o.token, "token", "", "Long-lived access token (omit to prompt)")
	cmd.Flags().BoolVar(&o.tokenStdin, "token-stdin", false, "Read the token from stdin")
	cmd.Flags().BoolVar(&o.insecure, "insecure", false, "Skip TLS certificate verification")
	cmd.Flags().IntVar(&o.timeout, "timeout", 0, "Request timeout in seconds (default 10)")
	cmd.Flags().BoolVar(&o.force, "force", false, "Overwrite an existing profile that still has a valid token")
	return cmd
}

const manualSetupText = `hass-cli init needs a terminal to prompt interactively.
On a non-interactive shell, configure it one of these ways:

  # Save a profile (token via stdin, stored in the OS keychain):
  printf '%s' "$HASS_TOKEN" | hass-cli profile login home \
      --server http://homeassistant.local:8123 --token-stdin

  # Or use environment variables (no keychain):
  export HASS_SERVER=http://homeassistant.local:8123
  export HASS_TOKEN=<long-lived access token>

Create a token in HA: profile page -> Long-Lived Access Tokens -> Create Token.
`

func printManualSetup(cmd *cobra.Command) {
	_, _ = io.WriteString(cmd.OutOrStdout(), manualSetupText)
}
