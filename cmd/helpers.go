package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bytetrade/hass-cli/internal/client"
	"github.com/bytetrade/hass-cli/internal/cmdutil"
	"github.com/bytetrade/hass-cli/internal/output"
	"github.com/spf13/cobra"
)

// Default table columns for list commands, used when the user does not pass
// --columns. JSON/YAML output ignores these.
const (
	colsState       = "ENTITY=entity_id,STATE=state,NAME=attributes.friendly_name"
	colsIntegration = "ID=entry_id,DOMAIN=domain,TITLE=title,STATE=state"
	colsHelper      = "ID=id,NAME=name"
	colsStatistics  = "ID=statistic_id,UNIT=unit_of_measurement,SOURCE=source"
	colsDashboard   = "ID=id,TITLE=title,URL=url_path,MODE=mode"
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

// errNoSupervisor is returned when a Supervisor-only command runs against a
// Core/Container install that has no Supervisor.
var errNoSupervisor = errors.New("this Home Assistant instance has no Supervisor (not a HA OS/Supervised install); add-ons and Supervisor management are unavailable here")

// requireSupervisor returns a connected client only if the target instance is a
// Supervised/HA OS install, so add-on commands fail fast with a clear message.
func requireSupervisor(cmd *cobra.Command, f *cmdutil.Factory) (*client.Client, error) {
	c, err := f.Client()
	if err != nil {
		return nil, err
	}
	ok, err := c.HasSupervisor(cmd.Context())
	if err != nil {
		c.Close()
		return nil, err
	}
	if !ok {
		c.Close()
		return nil, errNoSupervisor
	}
	return c, nil
}

// wsCall issues a single WS command with an already-built payload and renders
// the result. Use it when the payload needs imperative construction or
// validation that wsRun's declarative form can't express.
func wsCall(f *cmdutil.Factory, cmd *cobra.Command, payload map[string]any) error {
	c, err := f.Client()
	if err != nil {
		return err
	}
	defer c.Close()
	raw, err := c.WS(cmd.Context(), payload)
	if err != nil {
		return err
	}
	return renderRaw(f, raw)
}

// wsCallCols is wsCall with command-supplied default table columns.
func wsCallCols(f *cmdutil.Factory, cmd *cobra.Command, payload map[string]any, cols string) error {
	c, err := f.Client()
	if err != nil {
		return err
	}
	defer c.Close()
	raw, err := c.WS(cmd.Context(), payload)
	if err != nil {
		return err
	}
	return renderRawCols(f, raw, cols)
}

// wsRunCols is wsRun with command-supplied default table columns.
func wsRunCols(f *cmdutil.Factory, cols string, payload func(args []string) map[string]any) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return wsCallCols(f, cmd, payload(args), cols)
	}
}

// supervisorRun returns a cobra RunE that proxies a Supervisor endpoint, gated
// on a Supervised/HA OS install via requireSupervisor.
func supervisorRun(f *cmdutil.Factory, method string, endpoint func(args []string) string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := requireSupervisor(cmd, f)
		if err != nil {
			return err
		}
		defer c.Close()
		raw, err := c.SupervisorAPI(cmd.Context(), method, endpoint(args), nil)
		if err != nil {
			return err
		}
		return renderRaw(f, raw)
	}
}

// requireData parses --data and fails if it is empty, so write commands report
// a clear local error instead of sending an empty payload to Home Assistant.
func requireData(data string) (map[string]any, error) {
	m, err := parseDataObject(data)
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, errors.New("--data is required (a non-empty JSON object, or @file.json)")
	}
	return m, nil
}

// requireDataField is like requireData but also requires a specific field to be
// present (e.g. "name" on a create).
func requireDataField(data, field string) (map[string]any, error) {
	m, err := requireData(data)
	if err != nil {
		return nil, err
	}
	if _, ok := m[field]; !ok {
		return nil, fmt.Errorf("--data must include %q", field)
	}
	return m, nil
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

// renderRawCols is renderRaw with command-supplied default table columns. The
// defaults apply only to table output and only when the user did not pass
// --columns, so explicit user columns and non-table formats are unaffected.
func renderRawCols(f *cmdutil.Factory, raw json.RawMessage, defaultCols string) error {
	r, err := f.Renderer()
	if err != nil {
		return err
	}
	if f.Columns == "" && defaultCols != "" {
		r.Columns = output.ParseColumns(defaultCols)
	}
	var v any
	if len(raw) == 0 {
		return r.Render(map[string]any{"success": true})
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return r.Render(string(raw))
	}
	return r.Render(v)
}

// renderValueCols is renderValue with command-supplied default table columns.
func renderValueCols(f *cmdutil.Factory, v any, defaultCols string) error {
	r, err := f.Renderer()
	if err != nil {
		return err
	}
	if f.Columns == "" && defaultCols != "" {
		r.Columns = output.ParseColumns(defaultCols)
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
