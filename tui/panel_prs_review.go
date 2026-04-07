package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MichaelRBond/devdash/types"
)

// PRsReviewMsg carries fetched review PRs.
type PRsReviewMsg struct {
	Items []types.PR
	Err   error
}

// PanelPRsReview displays PRs requesting the user's review.
type PanelPRsReview struct {
	items    []types.PR
	loading  bool
	err      error
	focused  bool
	selected int
	width    int
	height   int
	styles   Styles
}

func NewPanelPRsReview(styles Styles) PanelPRsReview {
	return PanelPRsReview{loading: true, styles: styles}
}

func (p PanelPRsReview) Update(msg tea.Msg) (PanelPRsReview, tea.Cmd) {
	switch msg := msg.(type) {
	case PRsReviewMsg:
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
			if len(p.items) > 0 && p.selected < len(p.items) {
				openURL(p.items[p.selected].URL)
			}
		}
	}
	return p, nil
}

func (p PanelPRsReview) View() string {
	title := p.styles.Muted.Render("[3] ") + p.styles.PanelTitle.Render("PRs to Review")
	if p.focused {
		title = "▶ " + title
	}

	var content string
	switch {
	case p.loading:
		content = p.styles.Muted.Render("Loading...")
	case p.err != nil:
		content = p.styles.Danger.Render("Error: " + p.err.Error())
	case len(p.items) == 0:
		content = p.styles.Muted.Render("No PRs to review")
	default:
		var lines []string
		for i, pr := range p.items {
			age := styledAge(p.styles, pr.CreatedAt)
			review := reviewStatusIcon(p.styles, pr.ReviewStatus)
			line := fmt.Sprintf("%s %s #%d %s  %s  %s",
				review,
				p.styles.Muted.Render(repoName(pr.Repo)),
				pr.Number,
				truncate(pr.Title, 38),
				p.styles.Accent.Render(pr.Author),
				age,
			)
			if i == p.selected && p.focused {
				line = p.styles.Accent.Render("→ ") + line
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
		}
		// viewport = panel inner height - border(2) - title(1) - blank(1) - padding(1)
		viewport := p.height - 5
		content = scrollView(lines, p.selected, viewport)
	}

	panel := title + "\n\n" + content
	return RenderPanel(p.styles, panel, p.width, p.height, p.focused)
}

func (p *PanelPRsReview) SetSize(w, h int) { p.width = w; p.height = h }
func (p *PanelPRsReview) SetFocused(f bool) { p.focused = f }

// scrollView returns a slice of lines visible in the viewport, keeping
// the selected index visible. Returns the visible lines joined with newlines.
// viewportHeight is the number of lines that fit in the panel content area.
func scrollView(lines []string, selected, viewportHeight int) string {
	if viewportHeight <= 0 || len(lines) <= viewportHeight {
		return strings.Join(lines, "\n")
	}

	// Calculate scroll offset to keep selected visible.
	start := 0
	if selected >= viewportHeight {
		start = selected - viewportHeight + 1
	}
	if start+viewportHeight > len(lines) {
		start = len(lines) - viewportHeight
	}
	if start < 0 {
		start = 0
	}

	end := start + viewportHeight
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

func reviewStatusIcon(styles Styles, status types.ReviewStatus) string {
	switch status {
	case types.ReviewApproved:
		return styles.Success.Render("A")
	case types.ReviewChanges:
		return styles.Danger.Render("X")
	default:
		return styles.Muted.Render("-")
	}
}

// Shared helpers used by multiple panels.

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// styledAge returns the age string colored by urgency:
// green for ≤1d, yellow for 2-3d, red for >3d.
func styledAge(styles Styles, t time.Time) string {
	age := formatAge(t)
	days := int(time.Since(t).Hours() / 24)
	switch {
	case days <= 1:
		return styles.Success.Render(age)
	case days <= 3:
		return styles.Warning.Render(age)
	default:
		return styles.Danger.Render(age)
	}
}

// repoName strips the org prefix from "org/repo" → "repo".
func repoName(full string) string {
	if i := strings.LastIndex(full, "/"); i >= 0 {
		return full[i+1:]
	}
	return full
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}

func ciStatusIcon(styles Styles, status types.CIStatus) string {
	switch status {
	case types.CIStatusPassed:
		return styles.Success.Render("✓")
	case types.CIStatusFailed:
		return styles.Danger.Render("✗")
	case types.CIStatusPending:
		return styles.Warning.Render("●")
	default:
		return styles.Muted.Render("?")
	}
}
