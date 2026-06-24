// Package cmd assembles the hass-cli command tree.
package cmd

import (
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// version is overridden at build time via -ldflags.
var version = "dev"

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
		newSystemCmd(f),
		newRawCmds(f),
		newSkillCmd(),
	)

	return root
}
