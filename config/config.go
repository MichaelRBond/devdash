package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General  GeneralConfig  `toml:"general"`
	GitHub   GitHubConfig   `toml:"github"`
	Linear   LinearConfig   `toml:"linear"`
	Calendar CalendarConfig `toml:"calendar"`
	Weather  WeatherConfig  `toml:"weather"`
}

type GeneralConfig struct {
	RefreshInterval duration `toml:"refresh_interval"`
	Theme           string   `toml:"theme"`
	Layout          string   `toml:"layout"`
}


type GitHubConfig struct {
	Enabled         bool     `toml:"enabled"`
	TokenEnv        string   `toml:"token_env"`
	Orgs            []string `toml:"orgs"`
	Repos           []string `toml:"repos"`
	ReviewTeamSlugs []string `toml:"review_team_slugs"`
	OpenCommand     string   `toml:"open_command"`
}

type LinearConfig struct {
	Enabled          bool     `toml:"enabled"`
	TokenEnv         string   `toml:"token_env"`
	TeamKeys         []string `toml:"team_keys"`
	StatesUpNext     []string `toml:"states_up_next"`
	StatesInProgress []string `toml:"states_in_progress"`
	StatesInReview   []string `toml:"states_in_review"`
	OpenCommand      string   `toml:"open_command"`
}

type CalendarConfig struct {
	Enabled      bool     `toml:"enabled"`
	Provider     string   `toml:"provider"`
	CalendarIDs  []string `toml:"calendar_ids"`
	DaysAhead    int      `toml:"days_ahead"`
	ShowDeclined bool     `toml:"show_declined"`
	OpenCommand  string   `toml:"open_command"`
}

type WeatherConfig struct {
	Enabled  bool   `toml:"enabled"`
	Location string `toml:"location"`
	Units    string `toml:"units"` // "fahrenheit" | "celsius"
}

// duration wraps time.Duration for TOML string parsing (e.g. "5m").
type duration time.Duration

func (d *duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(text), err)
	}
	*d = duration(parsed)
	return nil
}

func Defaults() Config {
	return Config{
		General: GeneralConfig{
			RefreshInterval: duration(5 * time.Minute),
			Theme:           "dark",
			Layout:          "default",
		},
		GitHub: GitHubConfig{
			Enabled:  true,
			TokenEnv: "DEVDASH_GITHUB_TOKEN",
		},
		Linear: LinearConfig{
			Enabled:          true,
			TokenEnv:         "LINEAR_API_KEY",
			StatesUpNext:     []string{"Up Next", "Todo", "Ready"},
			StatesInProgress: []string{"In Progress"},
			StatesInReview:   []string{"In Review", "PR Review"},
		},
		Calendar: CalendarConfig{
			Enabled:     true,
			Provider:    "google",
			CalendarIDs: []string{"primary"},
			DaysAhead:   3,
		},
		Weather: WeatherConfig{
			Enabled:  true,
			Location: "",
			Units:    "fahrenheit",
		},
	}
}

// Load reads configuration from the given TOML file path.
// If the file does not exist, it returns defaults.
// If the file exists but is invalid, it returns an error.
func Load(path string) (Config, error) {
	cfg := Defaults()

	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

// ConfigDir returns the XDG-compliant config directory for devdash.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg + "/devdash"
	}
	home, _ := os.UserHomeDir()
	return home + "/.config/devdash"
}

// DefaultConfigPath returns the default path to the config file.
func DefaultConfigPath() string {
	return ConfigDir() + "/config.toml"
}
