package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MichaelRBond/devdash/config"
	"github.com/MichaelRBond/devdash/providers"
	"github.com/MichaelRBond/devdash/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("devdash", version)
			os.Exit(0)
		case "auth":
			if len(os.Args) > 2 && os.Args[2] == "google" {
				if err := providers.AuthGoogle(config.ConfigDir()); err != nil {
					fmt.Fprintf(os.Stderr, "auth error: %v\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "Usage: devdash auth google\n")
			os.Exit(1)
		}
	}

	cfg, err := config.Load(config.DefaultConfigPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	theme := tui.ThemeFromName(cfg.General.Theme)
	styles := tui.NewStyles(theme)

	if cfg.General.OpenCommand != "" {
		tui.SetOpenCommand(cfg.General.OpenCommand)
	}

	var ghProvider *providers.GitHubProvider
	if cfg.GitHub.Enabled {
		gp, err := providers.NewGitHubProvider(cfg.GitHub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "github: %v (panel disabled)\n", err)
		} else {
			ghProvider = gp
		}
	}

	var linProvider *providers.LinearProvider
	if cfg.Linear.Enabled {
		lp, err := providers.NewLinearProvider(cfg.Linear)
		if err != nil {
			fmt.Fprintf(os.Stderr, "linear: %v (panel disabled)\n", err)
		} else {
			linProvider = lp
		}
	}

	var claudeProvider *providers.ClaudeProvider
	if cfg.Claude.Enabled {
		claudeProvider = providers.NewClaudeProvider(cfg.Claude)
	}

	var calProvider *providers.CalendarProvider
	if cfg.Calendar.Enabled {
		cp, err := providers.NewCalendarProvider(cfg.Calendar, config.ConfigDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "calendar: %v (panel disabled)\n", err)
		} else {
			calProvider = cp
		}
	}

	app := tui.NewApp(styles, time.Duration(cfg.General.RefreshInterval), ghProvider, linProvider, claudeProvider, calProvider)

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
