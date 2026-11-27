package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSkillExport checks that export writes references and replaces a skill
// folder whole.
func TestSkillExport(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "ha-shared", "references", "gone.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "lares-media-create", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := runAll(t, "skill", "export", dir); err != nil {
		t.Fatalf("export: %v\n%s", err, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "ha-shared", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	refs, _ := filepath.Glob(filepath.Join(dir, "ha-shared", "references", "*.md"))
	if len(refs) == 0 {
		t.Fatal("export wrote no references")
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("a reference the build no longer has survived the export")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("export removed a folder that is not one of its skills")
	}
}
