package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all color tokens for the dashboard.
type Theme struct {
	Bg      lipgloss.Color
	PanelBg lipgloss.Color
	Accent  lipgloss.Color
	Warning lipgloss.Color
	Success lipgloss.Color
	Danger  lipgloss.Color
	Muted   lipgloss.Color
	Text    lipgloss.Color
}

var DarkTheme = Theme{
	Bg:      lipgloss.Color("#1a1b26"),
	PanelBg: lipgloss.Color("#24283b"),
	Accent:  lipgloss.Color("#7dcfff"),
	Warning: lipgloss.Color("#e0af68"),
	Success: lipgloss.Color("#9ece6a"),
	Danger:  lipgloss.Color("#f7768e"),
	Muted:   lipgloss.Color("#565f89"),
	Text:    lipgloss.Color("#c0caf5"),
}

var LightTheme = Theme{
	Bg:      lipgloss.Color("#f0f0f0"),
	PanelBg: lipgloss.Color("#ffffff"),
	Accent:  lipgloss.Color("#0077b6"),
	Warning: lipgloss.Color("#d4880f"),
	Success: lipgloss.Color("#2d7d46"),
	Danger:  lipgloss.Color("#c62828"),
	Muted:   lipgloss.Color("#888888"),
	Text:    lipgloss.Color("#1a1a1a"),
}

// Styles holds pre-computed lipgloss styles derived from a theme.
type Styles struct {
	Theme      Theme
	Panel      lipgloss.Style
	PanelTitle lipgloss.Style
	Accent     lipgloss.Style
	Warning    lipgloss.Style
	Success    lipgloss.Style
	Danger     lipgloss.Style
	Muted      lipgloss.Style
	StatusBar  lipgloss.Style
}

// NewStyles creates a Styles from the given theme.
func NewStyles(theme Theme) Styles {
	return Styles{
		Theme: theme,
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Muted),
		PanelTitle: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true),
		Accent:  lipgloss.NewStyle().Foreground(theme.Accent),
		Warning: lipgloss.NewStyle().Foreground(theme.Warning),
		Success: lipgloss.NewStyle().Foreground(theme.Success),
		Danger:  lipgloss.NewStyle().Foreground(theme.Danger),
		Muted:   lipgloss.NewStyle().Foreground(theme.Muted),
		StatusBar: lipgloss.NewStyle().
			Background(theme.PanelBg).
			Foreground(theme.Muted).
			Padding(0, 1),
	}
}

// RenderPanel renders content inside a bordered panel of exact outer dimensions.
// Border adds 2 to width and 2 to height. Content is placed top-left with 1-char margin.
func RenderPanel(styles Styles, content string, width, height int, focused bool) string {
	// Inner area = outer minus border (1 each side)
	innerW := width - 2
	innerH := height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	// Place content top-left with a small margin, clamped to inner width
	padded := lipgloss.NewStyle().Padding(0, 1).MaxWidth(innerW).Render(content)
	inner := lipgloss.Place(innerW, innerH, lipgloss.Left, lipgloss.Top, padded)

	border := styles.Panel
	if focused {
		border = border.BorderForeground(styles.Theme.Accent)
	}
	return border.Render(inner)
}

// ThemeFromName returns the theme for the given config value.
func ThemeFromName(name string) Theme {
	switch name {
	case "light":
		return LightTheme
	default:
		return DarkTheme
	}
}
