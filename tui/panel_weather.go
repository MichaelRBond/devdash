package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MichaelRBond/devdash/types"
)

// WeatherMsg carries fetched weather data.
type WeatherMsg struct {
	Weather types.Weather
	Err     error
}

// clockTickMsg fires every second for clock updates.
type clockTickMsg time.Time

// PanelWeather displays an ASCII clock and weather forecast.
type PanelWeather struct {
	weather    types.Weather
	unitSymbol string
	loading    bool
	err        error
	focused    bool
	width      int
	height     int
	styles     Styles
}

func NewPanelWeather(styles Styles, unitSymbol string) PanelWeather {
	return PanelWeather{loading: true, styles: styles, unitSymbol: unitSymbol}
}

func (p PanelWeather) Update(msg tea.Msg) (PanelWeather, tea.Cmd) {
	switch msg := msg.(type) {
	case WeatherMsg:
		p.loading = false
		p.err = msg.Err
		if msg.Err == nil {
			p.weather = msg.Weather
		}
	}
	return p, nil
}

func (p PanelWeather) View() string {
	title := p.styles.Muted.Render("[2] ") + p.styles.PanelTitle.Render("Clock & Weather")
	if p.focused {
		title = "▶ " + title
	}

	clock := p.renderClock()

	var weatherContent string
	switch {
	case p.loading:
		weatherContent = p.styles.Muted.Render("Loading weather...")
	case p.err != nil:
		weatherContent = p.styles.Danger.Render("Weather: " + p.err.Error())
	case !p.weather.Available:
		weatherContent = p.styles.Muted.Render("Weather not configured")
	default:
		weatherContent = p.renderWeather()
	}

	panel := title + "\n\n" + clock + "\n\n" + weatherContent
	return RenderPanel(p.styles, panel, p.width, p.height, p.focused)
}

func (p PanelWeather) renderClock() string {
	now := time.Now()
	hour := now.Hour()
	ampm := "AM"
	if hour >= 12 {
		ampm = "PM"
	}
	if hour > 12 {
		hour -= 12
	}
	if hour == 0 {
		hour = 12
	}
	min := now.Minute()

	h1, h2 := hour/10, hour%10
	m1, m2 := min/10, min%10

	var rows [3]string
	for row := 0; row < 3; row++ {
		parts := []string{}
		if h1 > 0 {
			parts = append(parts, digitRows[h1][row])
		}
		parts = append(parts, digitRows[h2][row])
		parts = append(parts, colonRows[row])
		parts = append(parts, digitRows[m1][row])
		parts = append(parts, digitRows[m2][row])
		rows[row] = strings.Join(parts, " ")
	}

	rows[1] = rows[1] + "  " + p.styles.Muted.Render(ampm)

	colored := make([]string, 3)
	for i, row := range rows {
		colored[i] = p.styles.Accent.Render(row)
	}

	return strings.Join(colored, "\n")
}

var digitRows = [10][3]string{
	{"█▀█", "█ █", "▀▀▀"}, // 0
	{" ▀█", "  █", "  ▀"}, // 1
	{"▀▀█", "█▀▀", "▀▀▀"}, // 2
	{"▀▀█", " ▀█", "▀▀▀"}, // 3
	{"█ █", "▀▀█", "  ▀"}, // 4
	{"█▀▀", "▀▀█", "▀▀▀"}, // 5
	{"█▀▀", "█▀█", "▀▀▀"}, // 6
	{"▀▀█", "  █", "  ▀"}, // 7
	{"█▀█", "█▀█", "▀▀▀"}, // 8
	{"█▀█", "▀▀█", "▀▀▀"}, // 9
}

var colonRows = [3]string{" ", "░", " "}

func (p PanelWeather) renderWeather() string {
	w := p.weather
	unit := p.unitSymbol

	current := fmt.Sprintf("%s %.0f%s %s",
		weatherIconStyled(p.styles, w.CurrentCode),
		w.CurrentTemp,
		unit,
		p.styles.Muted.Render(w.CurrentDesc),
	)

	var days []string
	for _, day := range w.Forecast {
		days = append(days, fmt.Sprintf("%s %s%.0f/%.0f%s",
			p.styles.Muted.Render(day.Date),
			weatherIconStyled(p.styles, day.Code),
			day.High, day.Low, unit,
		))
	}
	forecast := strings.Join(days, "   ")

	return current + "\n" + forecast
}

func weatherIconStyled(styles Styles, code int) string {
	icon := weatherIconChar(code) + " "
	switch {
	case code == 0:
		return styles.Warning.Render(icon)
	case code >= 1 && code <= 3:
		return styles.Muted.Render(icon)
	case code >= 61 && code <= 67, code >= 80 && code <= 82, code >= 51 && code <= 57:
		return styles.Accent.Render(icon)
	case code >= 71 && code <= 77, code >= 85 && code <= 86:
		return icon
	case code >= 95:
		return styles.Danger.Render(icon)
	default:
		return styles.Muted.Render(icon)
	}
}

func weatherIconChar(code int) string {
	switch {
	case code == 0:
		return "☀"
	case code >= 1 && code <= 3:
		return "⛅"
	case code == 45 || code == 48:
		return "🌫"
	case code >= 51 && code <= 57:
		return "🌦"
	case code >= 61 && code <= 67:
		return "🌧"
	case code >= 71 && code <= 77:
		return "❄"
	case code >= 80 && code <= 82:
		return "🌧"
	case code >= 85 && code <= 86:
		return "❄"
	case code >= 95 && code <= 99:
		return "⛈"
	default:
		return "?"
	}
}

func ClockTickCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}

func (p *PanelWeather) SetSize(w, h int) { p.width = w; p.height = h }
func (p *PanelWeather) SetFocused(f bool) { p.focused = f }
