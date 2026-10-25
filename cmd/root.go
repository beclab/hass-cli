// Package cmd assembles the hass-cli command tree.
package cmd

import (
	"fmt"
	"strings"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// version is overridden by main via SetVersion (which may inject a build-time
// value); this default keeps `--version` meaningful for `go run`/tests.
var version = "0.0.1"

// SetVersion lets main inject the build version.
func SetVersion(v string) {
	if v != "" {
		version = v
	}
}

// NewRootCommand builds the root command with global flags and all subcommands.
func NewRootCommand() *cobra.Command {
	f := &cmdutil.Factory{}

	root := &cobra.Command{
		Use:           "hass-cli",
		Short:         "Command-line interface for Home Assistant",
		Long:          "hass-cli talks to a local or remote Home Assistant instance over its REST and WebSocket APIs.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		// A service name used as a command (hass-cli light.turn_off ...) is the
		// commonest wrong guess; say what the right command is.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if strings.Count(args[0], ".") == 1 && !strings.HasPrefix(args[0], "-") {
				return fmt.Errorf("unknown command %q for \"hass-cli\"; to call that service: hass-cli service call %s --entity <entity_id>", args[0], args[0])
			}
			if near := cmd.SuggestionsFor(args[0]); len(near) > 0 {
				return fmt.Errorf("unknown command %q for \"hass-cli\"; did you mean %s?", args[0], strings.Join(near, ", "))
			}
			return fmt.Errorf("unknown command %q for \"hass-cli\"; run hass-cli --help for the commands", args[0])
		},
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&f.Server, "server", "s", "", "Home Assistant URL (env HASS_SERVER)")
	pf.StringVar(&f.Token, "token", "", "Long-lived access token (env HASS_TOKEN)")
	pf.StringVar(&f.Profile, "profile", "", "Named profile from config.yaml")
	pf.BoolVar(&f.Insecure, "insecure", false, "Skip TLS certificate verification")
	pf.IntVar(&f.Timeout, "timeout", 0, "Request timeout in seconds (default 10)")
	pf.StringVarP(&f.Output, "output", "o", "table", "Output format: json|yaml|table|ndjson")
	pf.StringVar(&f.Columns, "columns", "", "Table columns, e.g. ENTITY=entity_id,STATE=state")
	pf.StringVar(&f.SortBy, "sort-by", "", "Sort table rows by a gjson path")
	pf.BoolVar(&f.NoHeaders, "no-headers", false, "Do not print table headers")

	root.AddCommand(
		newVersionCmd(),
		newProfileCmd(f),
		newPingCmd(f),
		newConfigCmd(f),
		newStateCmd(f),
		newServiceCmd(f),
		newEventCmd(f),
		newTemplateCmd(f),
		newRegistryCmds(f),
		newHelperCmds(f),
		newWorkflowCmds(f),
		newIntegrationCmd(f),
		newBackupCmd(f),
		newLovelaceCmd(f),
		newAssistCmd(f),
		newEnergyCmd(f),
		newStatisticsCmd(f),
		newAddonCmd(f),
		newSupervisorCmd(f),
		newSystemCmd(f),
		newRawCmds(f),
		newSkillCmd(),
	)

	return root
}
