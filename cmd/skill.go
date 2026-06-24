package cmd

import (
	"fmt"
	"io/fs"
	"sort"

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
		Short: "List bundled skills",
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
			for _, n := range names {
				fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
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
