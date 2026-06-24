// Package output renders command results in the format selected by the global
// --output flag: json, yaml, table, or ndjson.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
)

// Format enumerates supported render formats.
type Format string

const (
	FormatJSON   Format = "json"
	FormatYAML   Format = "yaml"
	FormatTable  Format = "table"
	FormatNDJSON Format = "ndjson"
)

// ParseFormat validates a user-provided format string.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatJSON, FormatYAML, FormatTable, FormatNDJSON:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unknown output format %q (want json|yaml|table|ndjson)", s)
	}
}

// Column maps a display header to a gjson path evaluated against each row.
type Column struct {
	Header string
	Path   string
}

// Renderer carries the chosen format plus optional table shaping options.
type Renderer struct {
	Format    Format
	Columns   []Column
	SortBy    string
	NoHeaders bool
	Out       io.Writer
}

// Render writes v in the configured format. For table output v should be a
// slice of objects; non-table formats accept any JSON-serializable value.
func (r *Renderer) Render(v any) error {
	switch r.Format {
	case FormatJSON:
		enc := json.NewEncoder(r.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case FormatYAML:
		return yaml.NewEncoder(r.Out).Encode(v)
	case FormatNDJSON:
		return r.renderNDJSON(v)
	case FormatTable:
		return r.renderTable(v)
	default:
		enc := json.NewEncoder(r.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}

func (r *Renderer) renderNDJSON(v any) error {
	rows, err := toRows(v)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(r.Out)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) renderTable(v any) error {
	rows, err := toRows(v)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Fprintln(r.Out, "(no results)")
		return nil
	}

	cols := r.Columns
	if len(cols) == 0 {
		cols = inferColumns(rows)
	}

	if r.SortBy != "" {
		sort.SliceStable(rows, func(i, j int) bool {
			return gjson.GetBytes(rows[i], r.SortBy).String() < gjson.GetBytes(rows[j], r.SortBy).String()
		})
	}

	tw := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
	if !r.NoHeaders {
		headers := make([]string, len(cols))
		for i, c := range cols {
			headers[i] = c.Header
		}
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
	}
	for _, row := range rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = gjson.GetBytes(row, c.Path).String()
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	return tw.Flush()
}

// toRows normalizes v into a slice of JSON objects. A single object becomes a
// one-element slice.
func toRows(v any) ([]json.RawMessage, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "[") {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		return rows, nil
	}
	return []json.RawMessage{json.RawMessage(raw)}, nil
}

// inferColumns picks the first row's top-level keys as columns, scalars first.
func inferColumns(rows []json.RawMessage) []Column {
	result := gjson.ParseBytes(rows[0])
	var cols []Column
	result.ForEach(func(key, value gjson.Result) bool {
		if value.IsObject() || value.IsArray() {
			return true
		}
		cols = append(cols, Column{Header: strings.ToUpper(key.String()), Path: key.String()})
		return true
	})
	if len(cols) == 0 {
		cols = []Column{{Header: "VALUE", Path: "@this"}}
	}
	return cols
}

// ParseColumns parses a --columns spec like "ENTITY=entity_id,NAME=attributes.friendly_name".
func ParseColumns(spec string) []Column {
	if strings.TrimSpace(spec) == "" {
		return nil
	}
	var cols []Column
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if eq := strings.Index(part, "="); eq >= 0 {
			cols = append(cols, Column{Header: part[:eq], Path: part[eq+1:]})
		} else {
			cols = append(cols, Column{Header: strings.ToUpper(part), Path: part})
		}
	}
	return cols
}
