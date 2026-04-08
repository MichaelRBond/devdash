package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpEntry struct {
	key  string
	desc string
}

var helpEntries = []helpEntry{
	{"Tab / Shift+Tab", "Cycle focus between panels"},
	{"j / k / ↑ / ↓", "Scroll within focused panel"},
	{"Enter", "Open selected item in browser"},
	{"s", "Open skill menu for focused panel"},
	{"r", "Refresh all panels"},
	{"R", "Refresh focused panel only"},
	{"1-5", "Jump to panel by number"},
	{"?", "Toggle this help overlay"},
	{"q", "Quit"},
}

// RenderHelp renders the help overlay centered on screen.
func RenderHelp(styles Styles, width, height int) string {
	title := styles.PanelTitle.Render("Keyboard Shortcuts")

	var rows []string
	for _, e := range helpEntries {
		row := fmt.Sprintf("  %s  %s",
			styles.Accent.Render(fmt.Sprintf("%18s", e.key)),
			e.desc,
		)
		rows = append(rows, row)
	}

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + styles.Muted.Render("Press ? or Esc to close")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Theme.Accent).
		Padding(1, 2)

	box := border.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
