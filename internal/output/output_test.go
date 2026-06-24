package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	for _, s := range []string{"json", "yaml", "table", "ndjson"} {
		if _, err := ParseFormat(s); err != nil {
			t.Errorf("ParseFormat(%q) unexpected error: %v", s, err)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(\"xml\") expected error, got nil")
	}
}

func TestParseColumns(t *testing.T) {
	cols := ParseColumns("ENTITY=entity_id, NAME=attributes.friendly_name")
	if len(cols) != 2 {
		t.Fatalf("want 2 columns, got %d", len(cols))
	}
	if cols[0].Header != "ENTITY" || cols[0].Path != "entity_id" {
		t.Errorf("col0 = %+v", cols[0])
	}
	if cols[1].Path != "attributes.friendly_name" {
		t.Errorf("col1 path = %q", cols[1].Path)
	}

	// Bare token: header derived by upper-casing the path.
	bare := ParseColumns("state")
	if len(bare) != 1 || bare[0].Header != "STATE" || bare[0].Path != "state" {
		t.Errorf("bare = %+v", bare)
	}

	// Empty / whitespace specs yield nil.
	if ParseColumns("") != nil || ParseColumns("   ") != nil {
		t.Error("empty spec should yield nil columns")
	}

	// Trailing/empty segments are skipped.
	if got := ParseColumns("A=a,,B=b"); len(got) != 2 {
		t.Errorf("want 2 columns ignoring empty segment, got %d", len(got))
	}
}

func TestToRows(t *testing.T) {
	// Array input -> one row per element.
	rows, err := toRows([]map[string]any{{"a": 1}, {"a": 2}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("array: want 2 rows, got %d", len(rows))
	}

	// Single object -> one row.
	rows, err = toRows(map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("object: want 1 row, got %d", len(rows))
	}

	// Bare scalar -> one row.
	rows, err = toRows("hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("scalar: want 1 row, got %d", len(rows))
	}
}

func render(t *testing.T, r *Renderer, v any) string {
	t.Helper()
	var buf bytes.Buffer
	r.Out = &buf
	if err := r.Render(v); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	return buf.String()
}

func TestRenderTableEmpty(t *testing.T) {
	r := &Renderer{Format: FormatTable}
	out := render(t, r, []any{})
	if !strings.Contains(out, "(no results)") {
		t.Errorf("empty array should print (no results), got %q", out)
	}
}

func TestRenderTableNullAndNested(t *testing.T) {
	r := &Renderer{
		Format: FormatTable,
		Columns: []Column{
			{Header: "NAME", Path: "name"},
			{Header: "NESTED", Path: "attrs.color"},
			{Header: "MISSING", Path: "nope"},
		},
	}
	rows := []map[string]any{
		{"name": "a", "attrs": map[string]any{"color": "red"}, "extra": nil},
		{"name": "b", "attrs": map[string]any{}},
	}
	out := render(t, r, rows)
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "NESTED") {
		t.Errorf("missing headers: %q", out)
	}
	// Nested path resolves; missing/null render as empty cells (no panic).
	if !strings.Contains(out, "red") {
		t.Errorf("nested path should resolve to red: %q", out)
	}
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Errorf("rows missing: %q", out)
	}
}

func TestRenderTableInferColumns(t *testing.T) {
	r := &Renderer{Format: FormatTable}
	rows := []map[string]any{{"id": "x1", "state": "on", "attrs": map[string]any{"k": "v"}}}
	out := render(t, r, rows)
	// Scalars are inferred as headers; nested object is skipped.
	if !strings.Contains(out, "ID") || !strings.Contains(out, "STATE") {
		t.Errorf("inferred headers missing: %q", out)
	}
	if strings.Contains(out, "ATTRS") {
		t.Errorf("nested object should not become a column: %q", out)
	}
}

func TestRenderTableSortBy(t *testing.T) {
	r := &Renderer{
		Format:  FormatTable,
		Columns: []Column{{Header: "ID", Path: "id"}},
		SortBy:  "id",
	}
	rows := []map[string]any{{"id": "c"}, {"id": "a"}, {"id": "b"}}
	out := render(t, r, rows)
	ai := strings.Index(out, "a")
	bi := strings.Index(out, "b")
	ci := strings.Index(out, "c")
	if !(ai < bi && bi < ci) {
		t.Errorf("rows not sorted ascending by id: %q", out)
	}
}

func TestRenderTableNoHeaders(t *testing.T) {
	r := &Renderer{
		Format:    FormatTable,
		Columns:   []Column{{Header: "ID", Path: "id"}},
		NoHeaders: true,
	}
	out := render(t, r, []map[string]any{{"id": "x1"}})
	if strings.Contains(out, "ID") {
		t.Errorf("--no-headers should omit header row: %q", out)
	}
	if !strings.Contains(out, "x1") {
		t.Errorf("data row missing: %q", out)
	}
}

func TestRenderNDJSON(t *testing.T) {
	r := &Renderer{Format: FormatNDJSON}
	out := render(t, r, []map[string]any{{"a": 1}, {"a": 2}})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 ndjson lines, got %d: %q", len(lines), out)
	}
}
