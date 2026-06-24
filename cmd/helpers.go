package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
)

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
