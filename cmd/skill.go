package cmd

import (
	"fmt"
	"io/fs"
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

	return cmd
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
