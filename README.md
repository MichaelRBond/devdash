# devdash

A terminal-based developer dashboard that consolidates calendar events, GitHub pull requests, Linear tasks, and Claude Code usage into a single TUI.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for developers who live in tmux/zellij.

```
┌──────────────────────────────────────────────┐
│  Upcoming Events                             │
├──────────────────────┬───────────────────────┤
│  PRs to Review       │  My PRs               │
├──────────────────────┼───────────────────────┤
│  Linear Tasks        │  Claude Code          │
└──────────────────────┴───────────────────────┘
```

## Install

```bash
go install github.com/MichaelRBond/devdash@latest
```

Or build from source:

```bash
git clone https://github.com/MichaelRBond/devdash.git
cd mbond-tui-dashboard
just build
./bin/devdash
```

## Setup

### GitHub PRs

Set a [classic personal access token](https://github.com/settings/tokens/new) with `repo` and `read:org` scopes:

```bash
export DEVDASH_GITHUB_TOKEN=ghp_...
```

> Uses `DEVDASH_GITHUB_TOKEN` instead of `GITHUB_TOKEN` to avoid conflicts with the `gh` CLI.

> Fine-grained tokens do not work with the GraphQL search API for private repos.

### Linear Tasks

Set your [Linear API key](https://linear.app/settings/api):

```bash
export LINEAR_API_KEY=lin_api_...
```

### Google Calendar

1. [Enable the Google Calendar API](https://console.cloud.google.com/apis/library/calendar-json.googleapis.com) for your Google Cloud project
2. Create an OAuth 2.0 Client ID (Desktop type) in the [Credentials page](https://console.cloud.google.com/apis/credentials)
3. If your OAuth consent screen is in "Testing" mode, add your Google account as a test user under [OAuth consent screen](https://console.cloud.google.com/apis/credentials/consent)
4. Save the credentials file:

```bash
mkdir -p ~/.config/devdash
# Save the downloaded JSON as:
~/.config/devdash/google-credentials.json
```

5. Run the auth flow:

```bash
devdash auth google
```

This opens your browser for authorization and saves the refresh token to `~/.config/devdash/google-token.json`.

### Weather

Displays an ASCII clock and 3-day weather forecast from [Open-Meteo](https://open-meteo.com/) (free, no API key needed).

```toml
[weather]
enabled = true
location = "Austin, TX"
units = "fahrenheit"    # "fahrenheit" | "celsius"
```

## Configuration

Config file: `~/.config/devdash/config.toml`

All panels are enabled by default. Panels gracefully degrade if their provider isn't configured.

```toml
[general]
refresh_interval = "5m"
theme = "dark"              # "dark" | "light"

[github]
enabled = true
token_env = "DEVDASH_GITHUB_TOKEN"
orgs = ["your-org"]         # filter by org (empty = all)
repos = []                  # filter by repo (empty = all in org)

[linear]
enabled = true
token_env = "LINEAR_API_KEY"
team_keys = ["PE"]
states_up_next = ["Up Next", "Todo", "Ready"]
states_in_progress = ["In Progress"]
states_in_review = ["In Review"]

[calendar]
enabled = true
calendar_ids = ["primary"]
days_ahead = 3
show_declined = false

[weather]
enabled = true
location = "Austin, TX"
units = "fahrenheit"
```

## Keybindings

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Cycle focus between panels |
| `j` / `k` | Scroll within focused panel |
| `Enter` | Open selected item in browser |
| `r` | Refresh all panels |
| `R` | Refresh focused panel only |
| `1`-`5` | Jump to panel by number |
| `?` | Show help overlay |
| `c` | Toggle compact/expanded view |
| `q` | Quit |

## Development

Requires Go 1.22+ and [just](https://github.com/casey/just).

```bash
just run          # run the dashboard
just build        # build to bin/devdash
just test         # run tests
just test-v       # run tests verbose
just test-race    # run tests with race detector
just fmt          # format code
just vet          # run go vet
just lint         # run golangci-lint
just check        # run all quality checks
just pre-commit   # fmt + check + test
```

## License

MIT
