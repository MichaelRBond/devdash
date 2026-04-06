# Bubble Tea Patterns for This Project

## Creating a New Panel

Every panel follows this structure:

1. Define a model struct in `tui/panel_<name>.go`
2. Implement `Init() tea.Cmd`, `Update(tea.Msg) (tea.Model, tea.Cmd)`, `View() string`
3. Define a `tea.Msg` type for data arrival (e.g., `LinearTasksMsg`)
4. Create a fetch command that calls the provider and returns the msg
5. Register the panel in `tui/app.go`

Template:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type PanelFoo struct {
    data    []types.Foo
    loading bool
    err     error
    focused bool
    width   int
    height  int
}

type FooDataMsg struct {
    Items []types.Foo
    Err   error
}

func NewPanelFoo() PanelFoo {
    return PanelFoo{loading: true}
}

func (m PanelFoo) Init() tea.Cmd {
    return m.fetchData
}

func (m PanelFoo) Update(msg tea.Msg) (PanelFoo, tea.Cmd) {
    switch msg := msg.(type) {
    case FooDataMsg:
        m.loading = false
        m.err = msg.Err
        m.data = msg.Items
    }
    return m, nil
}

func (m PanelFoo) View() string {
    if m.loading {
        return "Loading..."
    }
    if m.err != nil {
        return "Error: " + m.err.Error()
    }
    // render m.data with lipgloss
    return ""
}

func (m PanelFoo) fetchData() tea.Msg {
    // call provider, return FooDataMsg
    return FooDataMsg{}
}
```

## Async Data Fetching

Never block in Update. Always return a `tea.Cmd`:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case RefreshMsg:
        return m, tea.Batch(
            m.calendar.fetchData,
            m.prsReview.fetchData,
            m.prsMine.fetchData,
            m.linear.fetchData,
            m.claude.fetchData,
        )
    }
}
```

## Layout Composition

Use lipgloss.JoinHorizontal and JoinVertical to compose panels:

```go
func (m Model) View() string {
    top := lipgloss.JoinHorizontal(lipgloss.Top, m.calendar.View())
    middle := lipgloss.JoinHorizontal(lipgloss.Top, m.prsReview.View(), m.prsMine.View())
    bottom := lipgloss.JoinHorizontal(lipgloss.Top, m.linear.View(), m.claude.View())
    return lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)
}
```
