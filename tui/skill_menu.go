package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SkillMenu is an overlay that shows available skills for a panel.
type SkillMenu struct {
	skills   []Skill
	selected int
	floating bool // true = floating pane, false = new tab
	visible  bool
	styles   Styles
}

// NewSkillMenu creates a new skill menu (initially hidden).
func NewSkillMenu(styles Styles) SkillMenu {
	return SkillMenu{styles: styles, floating: true}
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

	footer := m.styles.Muted.Render("Enter: run  |  f: toggle mode  |  Esc: cancel") + "\n" + mode

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + footer

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Theme.Accent).
		Padding(1, 2)

	box := border.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
