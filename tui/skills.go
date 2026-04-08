package tui

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a discovered Claude Code skill.
type Skill struct {
	Name        string // full directory name, e.g. "github-review-pr"
	DisplayName string // prefix stripped, e.g. "review-pr"
	Path        string // full path to skill directory
}

// panelSkillPrefix maps panel index to skill directory prefix.
var panelSkillPrefix = map[int]string{
	panelCalendar:  "calendar",
	panelPRsReview: "github",
	panelPRsMine:   "github",
	panelLinear:    "linear",
}

// DiscoverSkills scans ~/.claude/skills/ for skills matching the given panel.
// Returns nil if the panel has no skill prefix or no skills are found.
func DiscoverSkills(panelIndex int) []Skill {
	prefix, ok := panelSkillPrefix[panelIndex]
	if !ok || prefix == "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	skillsDir := filepath.Join(home, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix+"-") {
			continue
		}
		// Check that SKILL.md exists in the directory.
		skillFile := filepath.Join(skillsDir, name, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		displayName := strings.TrimPrefix(name, prefix+"-")
		skills = append(skills, Skill{
			Name:        name,
			DisplayName: displayName,
			Path:        filepath.Join(skillsDir, name),
		})
	}

	return skills
}
