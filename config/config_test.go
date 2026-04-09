package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if time.Duration(cfg.General.RefreshInterval) != 5*time.Minute {
		t.Errorf("expected refresh interval 5m, got %v", cfg.General.RefreshInterval)
	}
	if cfg.General.Theme != "dark" {
		t.Errorf("expected theme dark, got %s", cfg.General.Theme)
	}
	if !cfg.GitHub.Enabled {
		t.Error("expected github enabled by default")
	}
	if cfg.GitHub.TokenEnv != "DEVDASH_GITHUB_TOKEN" {
		t.Errorf("expected token env DEVDASH_GITHUB_TOKEN, got %s", cfg.GitHub.TokenEnv)
	}
	if !cfg.Linear.Enabled {
		t.Error("expected linear enabled by default")
	}
	if cfg.Linear.TokenEnv != "LINEAR_API_KEY" {
		t.Errorf("expected token env LINEAR_API_KEY, got %s", cfg.Linear.TokenEnv)
	}
	if !cfg.Calendar.Enabled {
		t.Error("expected calendar enabled by default")
	}
	if cfg.Calendar.DaysAhead != 3 {
		t.Errorf("expected days ahead 3, got %d", cfg.Calendar.DaysAhead)
	}
	if !cfg.Weather.Enabled {
		t.Error("expected weather enabled by default")
	}
	if cfg.Weather.Units != "fahrenheit" {
		t.Errorf("expected units fahrenheit, got %s", cfg.Weather.Units)
	}
}

func TestLoadFromTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	tomlContent := `
[general]
refresh_interval = "10m"
theme = "light"

[github]
enabled = true
token_env = "MY_GH_TOKEN"
orgs = ["my-org"]

[linear]
enabled = false

[calendar]
enabled = true
days_ahead = 7
calendar_ids = ["primary", "work@example.com"]

[weather]
enabled = true
location = "Austin, TX"
units = "celsius"
`
	if err := os.WriteFile(path, []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if time.Duration(cfg.General.RefreshInterval) != 10*time.Minute {
		t.Errorf("expected 10m, got %v", cfg.General.RefreshInterval)
	}
	if cfg.General.Theme != "light" {
		t.Errorf("expected light, got %s", cfg.General.Theme)
	}
	if cfg.GitHub.TokenEnv != "MY_GH_TOKEN" {
		t.Errorf("expected MY_GH_TOKEN, got %s", cfg.GitHub.TokenEnv)
	}
	if len(cfg.GitHub.Orgs) != 1 || cfg.GitHub.Orgs[0] != "my-org" {
		t.Errorf("expected orgs [my-org], got %v", cfg.GitHub.Orgs)
	}
	if cfg.Linear.Enabled {
		t.Error("expected linear disabled")
	}
	if cfg.Calendar.DaysAhead != 7 {
		t.Errorf("expected days ahead 7, got %d", cfg.Calendar.DaysAhead)
	}
	if cfg.Weather.Location != "Austin, TX" {
		t.Errorf("expected location Austin, TX, got %s", cfg.Weather.Location)
	}
	if cfg.Weather.Units != "celsius" {
		t.Errorf("expected units celsius, got %s", cfg.Weather.Units)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("missing file should not error, got: %v", err)
	}
	defaults := Defaults()
	if time.Duration(cfg.General.RefreshInterval) != time.Duration(defaults.General.RefreshInterval) {
		t.Error("expected defaults when file is missing")
	}
}

func TestLoadInvalidTOMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("not valid toml [[["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}
