package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MichaelRBond/devdash/types"
)

// LinearTasksMsg carries fetched Linear tasks.
type LinearTasksMsg struct {
	Items []types.Task
	Err   error
}

// visibleStateGroups defines which state groups are shown in the panel.
var visibleStateGroups = map[string]bool{
	"up_next":     true,
	"in_progress": true,
	"in_review":   true,
}

// PanelLinear displays Linear tasks grouped by workflow state in columns.
type PanelLinear struct {
	items       []types.Task // all fetched tasks
	visible     []types.Task // only tasks with a visible state group
	loading     bool
	err         error
	focused     bool
	selected    int
	width       int
	height      int
	styles      Styles
	openCommand string
}

func NewPanelLinear(styles Styles, openCommand string) PanelLinear {
	return PanelLinear{loading: true, styles: styles, openCommand: openCommand}
}

func (p PanelLinear) Update(msg tea.Msg) (PanelLinear, tea.Cmd) {
	switch msg := msg.(type) {
	case LinearTasksMsg:
		p.loading = false
		p.err = msg.Err
		p.items = msg.Items
		p.visible = filterVisible(p.items)
		if p.selected >= len(p.visible) {
			p.selected = max(0, len(p.visible)-1)
		}
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.visible)-1 {
				p.selected++
			}
		case "k", "up":
			if p.selected > 0 {
				p.selected--
			}
		case "enter":
			if len(p.visible) > 0 && p.selected < len(p.visible) {
				openURLWith(p.visible[p.selected].URL, p.openCommand)
			}
		}
	}
	return p, nil
}

func (p PanelLinear) View() string {
	title := p.styles.Muted.Render("[5] ") + p.styles.PanelTitle.Render("Linear Tasks")
	if p.focused {
		title = "▶ " + title
	}

	var content string
	switch {
	case p.loading:
		content = p.styles.Muted.Render("Loading...")
	case p.err != nil:
		content = p.styles.Danger.Render("Error: " + p.err.Error())
	case len(p.visible) == 0:
		content = p.styles.Muted.Render("No tasks")
	default:
		content = p.renderColumns()
	}

	panel := title + "\n\n" + content
	return RenderPanel(p.styles, panel, p.width, p.height, p.focused)
}

func (p PanelLinear) renderColumns() string {
	type column struct {
		label string
		key   string
		style func(strs ...string) string
	}
	columns := []column{
		{"Up Next", "up_next", p.styles.Success.Render},
		{"In Progress", "in_progress", p.styles.Danger.Render},
		{"PR Review", "in_review", p.styles.Warning.Render},
	}

	innerWidth := p.width - 8
	colWidth := innerWidth / len(columns)
	if colWidth < 20 {
		colWidth = 20
	}

	// Track index into p.visible for selection highlighting.
	visibleIdx := 0

	var renderedCols []string
	for _, col := range columns {
		header := col.style(col.label)

		var lines []string
		for _, task := range p.visible {
			if task.StateGroup != col.key {
				continue
			}
			age := formatAge(task.CreatedAt)
			labels := ""
			if len(task.Labels) > 0 {
				labels = p.styles.Muted.Render(" [" + strings.Join(task.Labels, ", ") + "]")
			}

			titleMaxLen := colWidth - len(task.Identifier) - len(age) - 6
			if titleMaxLen < 10 {
				titleMaxLen = 10
			}

			line := fmt.Sprintf("%s %s%s  %s",
				p.styles.Muted.Render(task.Identifier),
				truncate(task.Title, titleMaxLen),
				labels,
				p.styles.Muted.Render(age),
			)
			if visibleIdx == p.selected && p.focused {
				line = p.styles.Accent.Render("→ ") + line
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
			visibleIdx++
		}

		colContent := header + "\n" + strings.Join(lines, "\n")
		if len(lines) == 0 {
			colContent = header + "\n" + p.styles.Muted.Render("  none")
		}

		colStyle := lipgloss.NewStyle().Width(colWidth)
		renderedCols = append(renderedCols, colStyle.Render(colContent))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
}

// stateGroupOrder defines the display order for columns.
var stateGroupOrder = []string{"up_next", "in_progress", "in_review"}

// filterVisible returns only tasks with a displayed state group,
// sorted in column order (Up Next first, then In Progress, then PR Review).
func filterVisible(tasks []types.Task) []types.Task {
	var result []types.Task
	for _, group := range stateGroupOrder {
		for _, t := range tasks {
			if t.StateGroup == group {
				result = append(result, t)
			}
		}
	}
	return result
}

func (p *PanelLinear) SetSize(w, h int) { p.width = w; p.height = h }
func (p *PanelLinear) SetFocused(f bool) { p.focused = f }

// SelectedMetadata returns JSON-serializable metadata for the selected task.
func (p *PanelLinear) SelectedMetadata() map[string]any {
	if len(p.visible) == 0 || p.selected >= len(p.visible) {
		return nil
	}
	task := p.visible[p.selected]
	return map[string]any{
		"panel":      "linear",
		"identifier": task.Identifier,
		"title":      task.Title,
		"url":        task.URL,
		"team":       task.Team,
		"state":      task.State,
		"labels":     task.Labels,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}
}
