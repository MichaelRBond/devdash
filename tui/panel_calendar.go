package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MichaelRBond/devdash/types"
)

type CalendarEventsMsg struct {
	Items []types.Event
	Err   error
}

type PanelCalendar struct {
	items    []types.Event
	loading     bool
	err         error
	focused     bool
	selected    int
	width       int
	height      int
	styles      Styles
	openCommand string
}

func NewPanelCalendar(styles Styles, openCommand string) PanelCalendar {
	return PanelCalendar{loading: true, styles: styles, openCommand: openCommand}
}

func (p PanelCalendar) Update(msg tea.Msg) (PanelCalendar, tea.Cmd) {
	switch msg := msg.(type) {
	case CalendarEventsMsg:
		p.loading = false
		p.err = msg.Err
		p.items = msg.Items
		if p.selected >= len(p.items) && len(p.items) > 0 {
			p.selected = len(p.items) - 1
		}
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.items)-1 {
				p.selected++
			}
		case "k", "up":
			if p.selected > 0 {
				p.selected--
			}
		case "enter":
			todayItems := p.todayEvents()
			if len(todayItems) > 0 && p.selected < len(todayItems) {
				event := todayItems[p.selected]
				url := event.MeetingURL
				if url == "" {
					url = event.URL
				}
				if url != "" {
					openURLWith(url, p.openCommand)
				}
			}
		}
	}
	return p, nil
}

func (p PanelCalendar) View() string {
	title := p.styles.Muted.Render("[1] ") + p.styles.PanelTitle.Render("Today's Events")
	if p.focused {
		title = "▶ " + title
	}

	todayEvents := p.todayEvents()

	var content string
	switch {
	case p.loading:
		content = p.styles.Muted.Render("Loading...")
	case p.err != nil:
		content = p.styles.Danger.Render("Error: " + p.err.Error())
	case len(todayEvents) == 0:
		content = p.styles.Muted.Render("No events today")
	default:
		lines := p.renderEventLines(todayEvents)
		viewport := p.height - 5
		content = scrollView(lines, p.selected, viewport)
	}

	panel := title + "\n\n" + content
	return RenderPanel(p.styles, panel, p.width, p.height, p.focused)
}

// todayEvents filters items to only those starting today.
func (p PanelCalendar) todayEvents() []types.Event {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	var result []types.Event
	for _, e := range p.items {
		if !e.StartTime.Before(todayStart) && e.StartTime.Before(tomorrowStart) {
			result = append(result, e)
		}
	}
	return result
}

func (p PanelCalendar) renderEventLines(events []types.Event) []string {
	var lines []string

	for i, event := range events {
		timeStr := event.StartTime.Format("3:04 PM")
		statusIcon := eventStatusIcon(p.styles, event.Status)

		line := fmt.Sprintf("  %s %s  %s",
			statusIcon,
			p.styles.Muted.Render(timeStr),
			event.Title,
		)

		if i == p.selected && p.focused {
			line = p.styles.Accent.Render("→") + line[1:]
		}

		lines = append(lines, line)
	}

	return lines
}

func eventStatusIcon(styles Styles, status types.EventStatus) string {
	switch status {
	case types.EventAccepted:
		return styles.Success.Render("●")
	case types.EventTentative:
		return styles.Warning.Render("○")
	case types.EventDeclined:
		return styles.Muted.Render("✗")
	default:
		return styles.Muted.Render("?")
	}
}

func (p *PanelCalendar) SetSize(w, h int) { p.width = w; p.height = h }
func (p *PanelCalendar) SetFocused(f bool) { p.focused = f }
