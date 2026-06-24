// Command hass-cli is a command-line interface for Home Assistant.
package main

import (
	"fmt"
	"os"

	"github.com/bytetrade/hass-cli/cmd"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.SetVersion(version)
	root := cmd.NewRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
