package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

var errAgentIDsRequired = errors.New(`--data is required (e.g. {"agent_ids":["backup.local"]}), or use --auto`)

var errLabsUpdateData = errors.New(`--data is required (e.g. {"domain":"...","preview_feature":"...","enabled":true})`)

// wsRun returns a cobra RunE that issues a single WS command whose payload is
// built from the positional args, then renders the result. It keeps the many
// read-only WS subcommands declarative.
func wsRun(f *cmdutil.Factory, payload func(args []string) map[string]any) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := f.Client()
		if err != nil {
			return err
		}
		defer c.Close()
		raw, err := c.WS(cmd.Context(), payload(args))
		if err != nil {
			return err
		}
		return renderRaw(f, raw)
	}
}

// parseDataObject decodes a --data flag into a JSON object. As a convenience
// (and to sidestep shell quoting of inline JSON, especially on PowerShell), a
// value of the form "@path" reads the JSON from a file. An empty value yields
// a nil map.
func parseDataObject(data string) (map[string]any, error) {
	if data == "" {
		return nil, nil
	}
	raw, err := readDataArg(data)
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("invalid JSON (expected an object): %w", err)
	}
	return fields, nil
}

// readDataArg returns the raw bytes for a --data style argument, resolving a
// leading "@" to a file path.
func readDataArg(data string) ([]byte, error) {
	if strings.HasPrefix(data, "@") {
		b, err := os.ReadFile(strings.TrimPrefix(data, "@"))
		if err != nil {
			return nil, fmt.Errorf("read --data file: %w", err)
		}
		return b, nil
	}
	return []byte(data), nil
}

// renderRaw decodes a raw JSON payload and renders it via the factory's
// configured output format.
func renderRaw(f *cmdutil.Factory, raw json.RawMessage) error {
	r, err := f.Renderer()
	if err != nil {
		return err
	}
	var v any
	if len(raw) == 0 {
		return r.Render(map[string]any{"success": true})
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		// Not JSON; render as a string value.
		return r.Render(string(raw))
	}
	return r.Render(v)
}

// renderValue renders an already-decoded value.
func renderValue(f *cmdutil.Factory, v any) error {
	r, err := f.Renderer()
	if err != nil {
		return err
	}
	return r.Render(v)
}
