package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MichaelRBond/devdash/providers"
	"github.com/MichaelRBond/devdash/types"
)

const (
	panelCalendar  = 0
	panelClaude    = 1
	panelPRsReview = 2
	panelPRsMine   = 3
	panelLinear    = 4
	panelCount     = 5
)

// tickMsg fires on the refresh interval.
type tickMsg time.Time

// githubResultMsg is an internal message carrying both PR lists from a single API call.
type githubResultMsg struct {
	review []types.PR
	mine   []types.PR
	err    error
}

// App is the root Bubble Tea model.
type App struct {
	styles          Styles
	focused         int
	width           int
	height          int
	showHelp        bool
	compact         bool
	refreshInterval time.Duration
	spinner         spinner.Model

	// Panels
	calendar  PanelCalendar
	prsReview PanelPRsReview
	prsMine   PanelPRsMine
	linear    PanelLinear
	claude    PanelClaude

	// Providers (nil if not configured)
	githubProvider   *providers.GitHubProvider
	linearProvider   *providers.LinearProvider
	claudeProvider   *providers.ClaudeProvider
	calendarProvider *providers.CalendarProvider

	// Panel names for status bar display.
	panelNames []string

	// Notification tracking — stores keys of known items to detect new ones.
	knownReviewPRs map[int]bool  // PR number → seen
	knownEvents    map[string]bool // "startTime|title" → seen
	initialLoad    bool           // suppress notifications on first fetch
}

// NewApp creates the root application model.
func NewApp(styles Styles, refreshInterval time.Duration, ghProvider *providers.GitHubProvider, linProvider *providers.LinearProvider, claudeProvider *providers.ClaudeProvider, calProvider *providers.CalendarProvider) App {
	s := spinner.New()
	s.Spinner = spinner.Dot

	calendar := NewPanelCalendar(styles)
	prsReview := NewPanelPRsReview(styles)
	prsMine := NewPanelPRsMine(styles)
	linear := NewPanelLinear(styles)
	claude := NewPanelClaude(styles)

	// If providers are nil, mark panels as not loading.
	if calProvider == nil {
		calendar.loading = false
		calendar.err = fmt.Errorf("Calendar not configured")
	}
	if ghProvider == nil {
		prsReview.loading = false
		prsReview.err = fmt.Errorf("GitHub not configured")
		prsMine.loading = false
		prsMine.err = fmt.Errorf("GitHub not configured")
	}
	if linProvider == nil {
		linear.loading = false
		linear.err = fmt.Errorf("Linear not configured")
	}
	if claudeProvider == nil {
		claude.loading = false
		claude.usage = types.Usage{Available: false}
	}

	return App{
		styles:           styles,
		focused:          panelCalendar,
		refreshInterval:  refreshInterval,
		spinner:          s,
		calendar:         calendar,
		prsReview:        prsReview,
		prsMine:          prsMine,
		linear:           linear,
		claude:           claude,
		githubProvider:   ghProvider,
		linearProvider:   linProvider,
		claudeProvider:   claudeProvider,
		calendarProvider: calProvider,
		panelNames: []string{
			"Today's Events",
			"Claude Code",
			"PRs to Review",
			"My PRs",
			"Linear Tasks",
		},
		knownReviewPRs: make(map[int]bool),
		knownEvents:    make(map[string]bool),
		initialLoad:    true,
	}
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		a.spinner.Tick,
		a.tickCmd(),
	}

	if a.calendarProvider != nil {
		cmds = append(cmds, fetchCalendarCmd(a.calendarProvider))
	}
	if a.githubProvider != nil {
		cmds = append(cmds, fetchGitHubCmd(a.githubProvider))
	}
	if a.linearProvider != nil {
		cmds = append(cmds, fetchLinearCmd(a.linearProvider))
	}
	if a.claudeProvider != nil {
		cmds = append(cmds, fetchClaudeCmd(a.claudeProvider))
	}

	return tea.Batch(cmds...)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updatePanelSizes()
		return a, nil

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if panel := a.panelAtPosition(msg.X, msg.Y); panel >= 0 {
				a.focused = panel
				a.updatePanelFocus()
			}
		}
		return a, nil

	case CalendarEventsMsg:
		if msg.Err == nil {
			a.checkNewEvents(msg.Items)
		}
		var cmd tea.Cmd
		a.calendar, cmd = a.calendar.Update(msg)
		return a, cmd

	case githubResultMsg:
		if msg.err == nil {
			a.checkNewReviewPRs(msg.review)
		}
		var cmd1, cmd2 tea.Cmd
		a.prsReview, cmd1 = a.prsReview.Update(PRsReviewMsg{Items: msg.review, Err: msg.err})
		a.prsMine, cmd2 = a.prsMine.Update(PRsMineMsg{Items: msg.mine, Err: msg.err})
		return a, tea.Batch(cmd1, cmd2)

	case LinearTasksMsg:
		var cmd tea.Cmd
		a.linear, cmd = a.linear.Update(msg)
		return a, cmd

	case ClaudeUsageMsg:
		var cmd tea.Cmd
		a.claude, cmd = a.claude.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		// Help overlay captures all keys except ? and q.
		if a.showHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				a.showHelp = false
			}
			return a, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			a.focused = (a.focused + 1) % panelCount
			a.updatePanelFocus()
		case "shift+tab":
			a.focused = (a.focused - 1 + panelCount) % panelCount
			a.updatePanelFocus()
		case "1":
			a.focused = panelCalendar
			a.updatePanelFocus()
		case "2":
			a.focused = panelClaude
			a.updatePanelFocus()
		case "3":
			a.focused = panelPRsReview
			a.updatePanelFocus()
		case "4":
			a.focused = panelPRsMine
			a.updatePanelFocus()
		case "5":
			a.focused = panelLinear
			a.updatePanelFocus()
		case "?":
			a.showHelp = true
		case "c":
			a.compact = !a.compact
		case "r":
			cmds = append(cmds, a.refreshAllCmd()...)
			return a, tea.Batch(cmds...)
		case "R":
			cmd := a.refreshFocusedCmd()
			if cmd != nil {
				return a, cmd
			}
		default:
			// Forward key to focused panel.
			var cmd tea.Cmd
			switch a.focused {
			case panelCalendar:
				a.calendar, cmd = a.calendar.Update(msg)
			case panelPRsReview:
				a.prsReview, cmd = a.prsReview.Update(msg)
			case panelPRsMine:
				a.prsMine, cmd = a.prsMine.Update(msg)
			case panelLinear:
				a.linear, cmd = a.linear.Update(msg)
			case panelClaude:
				a.claude, cmd = a.claude.Update(msg)
			}
			if cmd != nil {
				return a, cmd
			}
		}
		return a, nil

	case tickMsg:
		cmds = append(cmds, a.tickCmd())
		cmds = append(cmds, a.refreshAllCmd()...)
		return a, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	if a.showHelp {
		return a.renderHelp()
	}

	return a.renderLayout()
}

func (a App) renderLayout() string {
	// Calculate panel dimensions.
	fullWidth := a.width
	halfWidth := fullWidth / 2
	// Reserve 1 line for status bar.
	availHeight := a.height - 1
	// Top row: calendar + claude side by side.
	topHeight := availHeight / 3
	if topHeight < 14 {
		topHeight = 14
	}
	remaining := availHeight - topHeight
	midHeight := remaining / 2
	botHeight := remaining - midHeight

	// Set sizes on all panels.
	a.calendar.SetSize(halfWidth, topHeight)
	a.claude.SetSize(fullWidth-halfWidth, topHeight)
	a.prsReview.SetSize(halfWidth, midHeight)
	a.prsMine.SetSize(fullWidth-halfWidth, midHeight)
	a.linear.SetSize(fullWidth, botHeight)

	// Update focus state.
	a.calendar.SetFocused(a.focused == panelCalendar)
	a.claude.SetFocused(a.focused == panelClaude)
	a.prsReview.SetFocused(a.focused == panelPRsReview)
	a.prsMine.SetFocused(a.focused == panelPRsMine)
	a.linear.SetFocused(a.focused == panelLinear)

	// Render each panel.
	top := lipgloss.JoinHorizontal(lipgloss.Top, a.calendar.View(), a.claude.View())
	mid := lipgloss.JoinHorizontal(lipgloss.Top, a.prsReview.View(), a.prsMine.View())
	bot := a.linear.View()

	body := lipgloss.JoinVertical(lipgloss.Left, top, mid, bot)

	statusBar := a.styles.StatusBar.
		Width(fullWidth).
		Render(fmt.Sprintf(" devdash  |  Tab: navigate  |  r: refresh  |  ?: help  |  q: quit  |  focused: %s", a.panelNames[a.focused]))

	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

func (a App) renderHelp() string {
	return RenderHelp(a.styles, a.width, a.height)
}

func (a App) tickCmd() tea.Cmd {
	return tea.Tick(a.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) updatePanelFocus() {
	a.calendar.SetFocused(a.focused == panelCalendar)
	a.prsReview.SetFocused(a.focused == panelPRsReview)
	a.prsMine.SetFocused(a.focused == panelPRsMine)
	a.linear.SetFocused(a.focused == panelLinear)
	a.claude.SetFocused(a.focused == panelClaude)
}

func (a *App) updatePanelSizes() {
	fullWidth := a.width
	halfWidth := fullWidth / 2
	availHeight := a.height - 1
	topHeight := availHeight / 3
	if topHeight < 14 {
		topHeight = 14
	}
	remaining := availHeight - topHeight
	midHeight := remaining / 2
	botHeight := remaining - midHeight

	a.calendar.SetSize(halfWidth, topHeight)
	a.claude.SetSize(fullWidth-halfWidth, topHeight)
	a.prsReview.SetSize(halfWidth, midHeight)
	a.prsMine.SetSize(fullWidth-halfWidth, midHeight)
	a.linear.SetSize(fullWidth, botHeight)
}

// panelAtPosition returns which panel index a screen coordinate falls in, or -1.
func (a App) panelAtPosition(x, y int) int {
	halfWidth := a.width / 2
	availHeight := a.height - 1
	topHeight := availHeight / 3
	if topHeight < 14 {
		topHeight = 14
	}
	remaining := availHeight - topHeight
	midHeight := remaining / 2

	// Top row: y < topHeight
	if y < topHeight {
		if x < halfWidth {
			return panelCalendar
		}
		return panelClaude
	}
	// Middle row: y < topHeight + midHeight
	if y < topHeight+midHeight {
		if x < halfWidth {
			return panelPRsReview
		}
		return panelPRsMine
	}
	// Bottom row
	return panelLinear
}

func (a App) refreshAllCmd() []tea.Cmd {
	var cmds []tea.Cmd
	if a.calendarProvider != nil {
		cmds = append(cmds, fetchCalendarCmd(a.calendarProvider))
	}
	if a.githubProvider != nil {
		cmds = append(cmds, fetchGitHubCmd(a.githubProvider))
	}
	if a.linearProvider != nil {
		cmds = append(cmds, fetchLinearCmd(a.linearProvider))
	}
	if a.claudeProvider != nil {
		cmds = append(cmds, fetchClaudeCmd(a.claudeProvider))
	}
	return cmds
}

func (a App) refreshFocusedCmd() tea.Cmd {
	switch a.focused {
	case panelCalendar:
		if a.calendarProvider != nil {
			return fetchCalendarCmd(a.calendarProvider)
		}
	case panelPRsReview, panelPRsMine:
		if a.githubProvider != nil {
			return fetchGitHubCmd(a.githubProvider)
		}
	case panelLinear:
		if a.linearProvider != nil {
			return fetchLinearCmd(a.linearProvider)
		}
	case panelClaude:
		if a.claudeProvider != nil {
			return fetchClaudeCmd(a.claudeProvider)
		}
	}
	return nil
}

func fetchCalendarCmd(provider *providers.CalendarProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		events, err := provider.Fetch(ctx)
		return CalendarEventsMsg{Items: events, Err: err}
	}
}

func fetchGitHubCmd(provider *providers.GitHubProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		review, mine, err := provider.FetchAll(ctx)
		return githubResultMsg{review: review, mine: mine, err: err}
	}
}

func fetchLinearCmd(provider *providers.LinearProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tasks, err := provider.Fetch(ctx)
		return LinearTasksMsg{Items: tasks, Err: err}
	}
}

func fetchClaudeCmd(provider *providers.ClaudeProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := provider.Fetch(ctx)
		if err != nil || len(items) == 0 {
			return ClaudeUsageMsg{Err: err}
		}
		return ClaudeUsageMsg{Usage: items[0]}
	}
}

// checkNewReviewPRs compares incoming PRs against known ones and plays a sound if new ones appear.
func (a *App) checkNewReviewPRs(prs []types.PR) {
	hasNew := false
	for _, pr := range prs {
		if !a.knownReviewPRs[pr.Number] {
			if !a.initialLoad {
				hasNew = true
			}
			a.knownReviewPRs[pr.Number] = true
		}
	}
	if hasNew {
		playNotificationSound()
	}
	// Mark initial load complete after first successful fetch.
	a.initialLoad = false
}

// checkNewEvents compares incoming events against known ones and plays a sound if new ones appear.
func (a *App) checkNewEvents(events []types.Event) {
	hasNew := false
	for _, e := range events {
		key := e.StartTime.Format(time.RFC3339) + "|" + e.Title
		if !a.knownEvents[key] {
			if !a.initialLoad {
				hasNew = true
			}
			a.knownEvents[key] = true
		}
	}
	if hasNew {
		playNotificationSound()
	}
}

// playNotificationSound plays a short alert sound.
func playNotificationSound() {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("afplay", "/System/Library/Sounds/Hero.aiff")
		_ = cmd.Start()
		return
	}
	// Fallback: terminal bell.
	fmt.Print("\a")
}
