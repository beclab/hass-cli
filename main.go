// Command hass-cli is a command-line interface for Home Assistant.
package main

import (
	"fmt"
	"os"

	"github.com/beclab/hass-cli/cmd"
	"github.com/beclab/hass-cli/internal/client"
)

// version is the default build version; override at build time via
// -ldflags "-X main.version=...".
var version = "0.0.1"

func main() {
	cmd.SetVersion(version)
	root := cmd.NewRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", client.FriendlyMessage(err))
		os.Exit(1)
	}
}
