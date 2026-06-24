package cmd

import (
	"encoding/json"

	"github.com/bytetrade/hass-cli/internal/cmdutil"
)

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
