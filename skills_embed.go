package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/bytetrade/hass-cli/cmd"
)

// skillsEmbedFS embeds each skill's agent-readable SKILL.md so the binary
// serves skill content matching its version. References are added to the
// pattern once a skill ships a references/ directory.
//
//go:embed skills/*/SKILL.md
var skillsEmbedFS embed.FS

func init() {
	sub, err := fs.Sub(skillsEmbedFS, "skills")
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: skills embed assembly failed:", err)
		return
	}
	cmd.SetEmbeddedSkillContent(sub)
}
