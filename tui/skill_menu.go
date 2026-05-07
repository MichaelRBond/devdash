package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// skillModels lists the model choices shown in the skill menu.
// First entry is the user's configured default (no --model flag passed).
// The rest are stable aliases that always resolve to the latest of each tier,
// so this list does not need updating when new model versions ship.
var skillModels = []string{"default", "opus", "sonnet", "haiku"}

// effortLevels lists the --effort values shown in the skill menu.
var effortLevels = []string{"low", "medium", "high", "xhigh", "max"}

const defaultEffortIdx = 1 // "medium"

// SkillMenu is an overlay that shows available skills for a panel.
type SkillMenu struct {
	skills    []Skill
	selected  int
	floating  bool // true = floating pane, false = new tab
	visible   bool
	styles    Styles
	modelIdx  int // index into skillModels
	effortIdx int // index into effortLevels
}

// NewSkillMenu creates a new skill menu (initially hidden).
func NewSkillMenu(styles Styles) SkillMenu {
	return SkillMenu{styles: styles, floating: true, effortIdx: defaultEffortIdx}
}

// Show populates the menu with skills and makes it visible.
// Returns false if there are no skills to show.
func (m *SkillMenu) Show(skills []Skill) bool {
	if len(skills) == 0 {
		return false
	}
	m.skills = skills
	m.selected = 0
	m.visible = true
	return true
}

// Hide closes the menu.
func (m *SkillMenu) Hide() {
	m.visible = false
	m.skills = nil
}

// IsVisible returns whether the menu is showing.
func (m *SkillMenu) IsVisible() bool {
	return m.visible
}

// IsFloating returns whether skills will launch in a floating pane.
func (m *SkillMenu) IsFloating() bool {
	return m.floating
}

// SelectedModel returns the model alias to pass to claude --model, or
// an empty string when the user's configured default should be used.
func (m *SkillMenu) SelectedModel() string {
	if m.modelIdx <= 0 || m.modelIdx >= len(skillModels) {
		return ""
	}
	return skillModels[m.modelIdx]
}

// SelectedEffort returns the effort level to pass to claude --effort.
func (m *SkillMenu) SelectedEffort() string {
	if m.effortIdx < 0 || m.effortIdx >= len(effortLevels) {
		return effortLevels[defaultEffortIdx]
	}
	return effortLevels[m.effortIdx]
}

// Selected returns the currently highlighted skill, or nil if menu is hidden.
func (m *SkillMenu) Selected() *Skill {
	if !m.visible || len(m.skills) == 0 {
		return nil
	}
	return &m.skills[m.selected]
}

// HandleKey processes a key event. Returns the selected skill if Enter was pressed, or nil.
// Returns (selected skill or nil, should close menu).
func (m *SkillMenu) HandleKey(key string) (*Skill, bool) {
	switch key {
	case "j", "down":
		if m.selected < len(m.skills)-1 {
			m.selected++
		}
		return nil, false
	case "k", "up":
		if m.selected > 0 {
			m.selected--
		}
		return nil, false
	case "f":
		m.floating = !m.floating
		return nil, false
	case "left":
		if m.modelIdx > 0 {
			m.modelIdx--
		}
		return nil, false
	case "right":
		if m.modelIdx < len(skillModels)-1 {
			m.modelIdx++
		}
		return nil, false
	case "e":
		m.effortIdx = (m.effortIdx + 1) % len(effortLevels)
		return nil, false
	case "enter":
		skill := m.Selected()
		m.Hide()
		return skill, true
	case "esc", "s", "q":
		m.Hide()
		return nil, true
	}
	return nil, false
}

// Render returns the skill menu overlay string, centered in the given dimensions.
func (m *SkillMenu) Render(width, height int) string {
	if !m.visible || len(m.skills) == 0 {
		return ""
	}

	title := m.styles.PanelTitle.Render("Skills")

	var rows []string
	for i, skill := range m.skills {
		line := fmt.Sprintf("  %s", skill.DisplayName)
		if i == m.selected {
			line = m.styles.Accent.Render("→ " + skill.DisplayName)
		}
		rows = append(rows, line)
	}

	modeLabel := "floating"
	if !m.floating {
		modeLabel = "new tab"
	}
	mode := fmt.Sprintf("mode: %s", m.styles.Accent.Render(modeLabel))

	modelParts := make([]string, len(skillModels))
	for i, name := range skillModels {
		if i == m.modelIdx {
			modelParts[i] = m.styles.Accent.Render("[" + name + "]")
		} else {
			modelParts[i] = m.styles.Muted.Render(" " + name + " ")
		}
	}
	modelLine := "model: " + strings.Join(modelParts, " ")

	effortLine := fmt.Sprintf("effort: %s", m.styles.Accent.Render(m.SelectedEffort()))

	footer := m.styles.Muted.Render("Enter: run  |  ←/→: model  |  e: effort  |  f: toggle mode  |  Esc: cancel") + "\n" + mode

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + modelLine + "\n" + effortLine + "\n" + footer

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Theme.Accent).
		Padding(1, 2)

	box := border.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
