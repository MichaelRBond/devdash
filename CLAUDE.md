# CLAUDE.md

## Project Overview

devdash is a terminal-based developer dashboard written in Go. It consolidates
calendar events, GitHub PRs, Linear tasks, and Claude Code usage into a single
TUI that runs inside tmux/zellij.

## Tech Stack

- **Language:** Go 1.22+
- **TUI framework:** Bubble Tea (charmbracelet/bubbletea) — Elm architecture
- **Styling:** Lip Gloss (charmbracelet/lipgloss)
- **Components:** Bubbles (charmbracelet/bubbles) — spinners, viewports, tables
- **Config:** TOML via BurntSushi/toml
- **APIs:** GitHub GraphQL, Linear GraphQL, Google Calendar v3

## Project Structure

- `main.go` — entrypoint, config loading, starts the Bubble Tea program
- `config/` — TOML config parsing, defaults, validation
- `tui/` — all TUI models and views, one file per panel
- `providers/` — API clients, one per data source (github, linear, calendar, claude)
- `types/` — shared domain types (Event, PR, Task, Usage)
- `internal/` — caching layer, helpers

## Architecture Patterns

- Each panel is a Bubble Tea `Model` with its own `Init`, `Update`, `View`.
- The root model in `tui/app.go` composes all panels and manages layout/focus.
- Providers fetch data asynchronously and return results as `tea.Msg` via `tea.Cmd`.
- Data is cached in-memory with per-provider TTLs (see `internal/cache.go`).
- No global state — all state lives in the root model or panel models.

## Commands

All commands use `just` (see justfile):

- `just run` — run the dashboard
- `just build` — build the binary to `bin/devdash`
- `just fmt` — format code
- `just vet` — run go vet
- `just lint` — run golangci-lint (via go run, no global install)
- `just test` — run tests
- `just check` — run all quality checks (fmt + vet + lint)
- `just pre-commit` — full check + tests before committing

## Conventions

- Use `gofmt` style. No tabs vs spaces debates — Go settles this.
- Error handling: return errors, don't panic. Wrap with `fmt.Errorf("context: %w", err)`.
- Naming: follow Go conventions — short receiver names, exported names for public API.
- Tests go in `_test.go` files alongside the code they test.
- Provider interfaces should be defined in `types/` so panels depend on interfaces, not concrete clients.
- Config values use environment variable overrides where noted (e.g., `$GITHUB_TOKEN`).

## Key Design Decisions

- Read-only in v1 — no mutations (no PR merges, no task transitions).
- Single binary, zero runtime dependencies.
- All API tokens come from env vars or config file, never hardcoded.
- Panels are independently refreshable and gracefully degrade if a provider fails.

## Things To Watch Out For

- Bubble Tea models must return `tea.Cmd` from `Update`, not call side effects directly.
- Lip Gloss `Width()` and `Height()` are for measuring, not setting — use `.Width(n)` style method.
- GitHub GraphQL has a 5000 points/hour rate limit — keep queries efficient.
- Linear API has a 1500 req/hour limit — cache aggressively.
- Google Calendar OAuth needs a refresh token flow — store tokens in XDG config dir.
