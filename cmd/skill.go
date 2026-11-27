package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// embeddedSkills holds the skill content tree wired in by main via
// SetEmbeddedSkillContent. It is nil for builds without embedded skills.
var embeddedSkills fs.FS

// SetEmbeddedSkillContent installs the embedded skills filesystem.
func SetEmbeddedSkillContent(f fs.FS) {
	embeddedSkills = f
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "List and print bundled agent skills",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List bundled skills with their descriptions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if embeddedSkills == nil {
				return fmt.Errorf("no skills embedded in this build")
			}
			entries, err := fs.ReadDir(embeddedSkills, ".")
			if err != nil {
				return err
			}
			var names []string
			for _, e := range entries {
				if e.IsDir() {
					names = append(names, e.Name())
				}
			}
			sort.Strings(names)

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			for _, n := range names {
				desc := ""
				if data, err := fs.ReadFile(embeddedSkills, n+"/SKILL.md"); err == nil {
					desc = truncate(frontMatterField(string(data), "description"), 100)
				}
				fmt.Fprintf(tw, "%s\t%s\n", n, desc)
			}
			return tw.Flush()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <name>",
		Short: "Print a skill's SKILL.md",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if embeddedSkills == nil {
				return fmt.Errorf("no skills embedded in this build")
			}
			data, err := fs.ReadFile(embeddedSkills, args[0]+"/SKILL.md")
			if err != nil {
				return fmt.Errorf("skill %q not found: %w", args[0], err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "export <dir>",
		Short: "Write every bundled skill, references included, under <dir>",
		Long: `Write every bundled skill under <dir>, one folder per skill, as compiled into
this binary. Each skill folder is replaced whole, so a reference a newer
release dropped does not linger. Other folders in <dir> are left alone.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if embeddedSkills == nil {
				return fmt.Errorf("no skills embedded in this build")
			}
			n, err := exportSkills(embeddedSkills, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %d skills to %s\n", n, args[0])
			return nil
		},
	})

	return cmd
}

// exportSkills writes each top-level skill folder of src under dir, replacing
// any folder of the same name, and returns how many skills it wrote.
func exportSkills(src fs.FS, dir string) (int, error) {
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		target := filepath.Join(dir, e.Name())
		if err := os.RemoveAll(target); err != nil {
			return n, err
		}
		err := fs.WalkDir(src, e.Name(), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			out := filepath.Join(dir, filepath.FromSlash(p))
			if d.IsDir() {
				return os.MkdirAll(out, 0o755)
			}
			data, err := fs.ReadFile(src, p)
			if err != nil {
				return err
			}
			return os.WriteFile(out, data, 0o644)
		})
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// frontMatterField extracts a scalar field from a SKILL.md YAML front-matter
// block (between the leading --- fences) without pulling in a YAML dependency.
// It handles single/double quoted and bare values on a single line.
func frontMatterField(content, field string) string {
	content = strings.TrimLeft(content, "\ufeff \t\r\n")
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	rest := content[len("---"):]
	end := strings.Index(rest, "\n---")
	if end >= 0 {
		rest = rest[:end]
	}
	prefix := field + ":"
	for _, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		val = strings.TrimSuffix(strings.TrimPrefix(val, `"`), `"`)
		val = strings.TrimSuffix(strings.TrimPrefix(val, "'"), "'")
		return val
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
