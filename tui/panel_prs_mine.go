package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MichaelRBond/devdash/types"
)

// PRsMineMsg carries fetched authored PRs.
type PRsMineMsg struct {
	Items []types.PR
	Err   error
}

// PanelPRsMine displays the user's open PRs.
type PanelPRsMine struct {
	items       []types.PR
	loading     bool
	err         error
	focused     bool
	selected    int
	width       int
	height      int
	styles      Styles
	openCommand string
}

func NewPanelPRsMine(styles Styles, openCommand string) PanelPRsMine {
	return PanelPRsMine{loading: true, styles: styles, openCommand: openCommand}
}

func (p PanelPRsMine) Update(msg tea.Msg) (PanelPRsMine, tea.Cmd) {
	switch msg := msg.(type) {
	case PRsMineMsg:
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
				openURLWith(p.items[p.selected].URL, p.openCommand)
			}
		}
	}
	return p, nil
}

func (p PanelPRsMine) View() string {
	title := p.styles.Muted.Render("[4] ") + p.styles.PanelTitle.Render("My PRs")
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
		content = p.styles.Muted.Render("No open PRs")
	default:
		var lines []string
		for i, pr := range p.items {
			ci := ciStatusIcon(p.styles, pr.CIStatus)
			review := reviewStatusIcon(p.styles, pr.ReviewStatus)
			age := styledAge(p.styles, pr.CreatedAt)
			comments := ""
			if pr.CommentCount > 0 {
				comments = fmt.Sprintf("  %d comments", pr.CommentCount)
			}
			line := fmt.Sprintf("%s%s %s #%d %s  %s%s",
				ci,
				review,
				p.styles.Muted.Render(repoName(pr.Repo)),
				pr.Number,
				truncate(pr.Title, 33),
				age,
				p.styles.Muted.Render(comments),
			)
			if i == p.selected && p.focused {
				line = p.styles.Accent.Render("→ ") + line
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
		}
		viewport := p.height - 5
		content = scrollView(lines, p.selected, viewport)
	}

	panel := title + "\n\n" + content
	return RenderPanel(p.styles, panel, p.width, p.height, p.focused)
}

func (p *PanelPRsMine) SetSize(w, h int) { p.width = w; p.height = h }
func (p *PanelPRsMine) SetFocused(f bool) { p.focused = f }

// SelectedMetadata returns JSON-serializable metadata for the selected PR.
func (p *PanelPRsMine) SelectedMetadata() map[string]any {
	if len(p.items) == 0 || p.selected >= len(p.items) {
		return nil
	}
	pr := p.items[p.selected]
	return map[string]any{
		"panel":         "github",
		"repo":          pr.Repo,
		"repo_name":     repoName(pr.Repo),
		"pr_number":     pr.Number,
		"pr_title":      pr.Title,
		"pr_url":        pr.URL,
		"author":        pr.Author,
		"branch":        pr.Branch,
		"created_at":    pr.CreatedAt.Format(time.RFC3339),
		"review_status": string(pr.ReviewStatus),
		"ci_status":     string(pr.CIStatus),
	}
}
