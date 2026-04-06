package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MichaelRBond/devdash/types"
)

// ClaudeUsageMsg carries fetched Claude usage data.
type ClaudeUsageMsg struct {
	Usage types.Usage
	Err   error
}

// PanelClaude displays Claude Code usage information.
type PanelClaude struct {
	usage   types.Usage
	loading bool
	err     error
	focused bool
	width   int
	height  int
	styles  Styles
}

func NewPanelClaude(styles Styles) PanelClaude {
	return PanelClaude{loading: true, styles: styles}
}

func (p PanelClaude) Update(msg tea.Msg) (PanelClaude, tea.Cmd) {
	switch msg := msg.(type) {
	case ClaudeUsageMsg:
		p.loading = false
		p.err = msg.Err
		if msg.Err == nil {
			p.usage = msg.Usage
		}
	}
	return p, nil
}

func (p PanelClaude) View() string {
	title := p.styles.Muted.Render("[2] ") + p.styles.PanelTitle.Render("Claude Code")
	if p.focused {
		title = "▶ " + title
	}

	var content string
	switch {
	case p.loading:
		content = p.styles.Muted.Render("Loading...")
	case p.err != nil:
		content = p.styles.Danger.Render("Error: " + p.err.Error())
	case !p.usage.Available:
		content = p.styles.Muted.Render("Usage data unavailable")
	default:
		content = p.renderUsage()
	}

	planLabel := strings.ToUpper(p.usage.Plan)
	if planLabel == "" {
		planLabel = "—"
	}
	header := title + "  " + p.styles.Muted.Render(planLabel+" Plan")

	panel := header + "\n\n" + content
	return RenderPanel(p.styles, panel, p.width, p.height, p.focused)
}

func (p PanelClaude) renderUsage() string {
	u := p.usage
	var lines []string

	now := time.Now()

	// Token usage and cost.
	if u.TokenLimit > 0 {
		pct := float64(u.TotalTokens) / float64(u.TokenLimit)
		bar := renderProgressBar(p.styles, pct, 20)
		lines = append(lines, fmt.Sprintf("Tokens:  %s / %s  %s  %d%%",
			formatTokensShort(u.TotalTokens), formatTokensShort(u.TokenLimit),
			bar, int(pct*100)))
	} else {
		lines = append(lines, fmt.Sprintf("Tokens:  %s", formatTokensShort(u.TotalTokens)))
	}
	if u.CostUSD > 0.01 {
		lines = append(lines, fmt.Sprintf("Cost:    $%.2f", u.CostUSD))
	}

	// Burn rate.
	if u.BurnRate > 0 {
		costRate := u.CostUSD / u.ActiveMinutes
		lines = append(lines, fmt.Sprintf("Rate:    %.1f tokens/min  |  $%.2f/min", u.BurnRate, costRate))
	}

	// Predictions.
	if u.BurnRate > 0 && u.TokenLimit > 0 && u.TotalTokens < u.TokenLimit {
		minutesLeft := float64(u.TokenLimit-u.TotalTokens) / u.BurnRate
		exhaustTime := now.Add(time.Duration(minutesLeft) * time.Minute)
		lines = append(lines, fmt.Sprintf("Runs out: %s", p.styles.Warning.Render(exhaustTime.Format("3:04 PM"))))
	} else if u.TokenLimit > 0 && u.TotalTokens >= u.TokenLimit {
		lines = append(lines, p.styles.Danger.Render("Token limit reached!"))
	}

	// Reset time.
	if !u.ResetAt.IsZero() {
		remaining := time.Until(u.ResetAt)
		resetStr := u.ResetAt.Format("3:04 PM")
		if remaining > 0 {
			lines = append(lines, fmt.Sprintf("Resets:   %s (%s)", resetStr, formatDuration(remaining)))
		} else {
			lines = append(lines, "Resets:   "+p.styles.Success.Render("now"))
		}
	}

	// Model distribution.
	if len(u.ModelStats) > 0 {
		lines = append(lines, "")
		for _, ms := range u.ModelStats {
			name := shortModelName(ms.Model)
			pct := float64(0)
			if u.TotalTokens > 0 {
				pct = float64(ms.TotalTokens) / float64(u.TotalTokens) * 100
			}
			costStr := ""
			if ms.CostUSD > 0.01 {
				costStr = fmt.Sprintf("  $%.2f", ms.CostUSD)
			}
			lines = append(lines, fmt.Sprintf("  %s  %s  %.0f%%%s",
				p.styles.Accent.Render(name),
				formatTokensShort(ms.TotalTokens),
				pct,
				p.styles.Muted.Render(costStr)))
		}
	}

	return strings.Join(lines, "\n")
}

// shortModelName returns a compact display name for a model.
func shortModelName(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "opus"):
		return "opus  "
	case strings.Contains(lower, "haiku"):
		return "haiku "
	case strings.Contains(lower, "sonnet"):
		return "sonnet"
	default:
		if len(model) > 12 {
			return model[:12]
		}
		return model
	}
}

func renderProgressBar(styles Styles, pct float64, width int) string {
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(pct * float64(width))
	empty := width - filled

	barColor := styles.Success
	if pct > 0.8 {
		barColor = styles.Danger
	} else if pct > 0.6 {
		barColor = styles.Warning
	}

	return barColor.Render(strings.Repeat("█", filled)) +
		styles.Muted.Render(strings.Repeat("░", empty))
}

// formatTokensShort returns a compact token count like "1.2M" or "340K".
func formatTokensShort(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%dK", n/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM tokens", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%dK tokens", n/1_000)
	default:
		return fmt.Sprintf("%d tokens", n)
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func (p *PanelClaude) SetSize(w, h int) { p.width = w; p.height = h }
func (p *PanelClaude) SetFocused(f bool) { p.focused = f }
