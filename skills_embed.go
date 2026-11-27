package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/beclab/hass-cli/cmd"
)

// skillsEmbedFS embeds each skill's SKILL.md and references so the binary can
// serve the whole suite offline.
//
//go:embed skills/*/SKILL.md skills/*/references/*.md
var skillsEmbedFS embed.FS

func init() {
	sub, err := fs.Sub(skillsEmbedFS, "skills")
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: skills embed assembly failed:", err)
		return
	}
	cmd.SetEmbeddedSkillContent(sub)
}
