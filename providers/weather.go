package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/MichaelRBond/devdash/config"
	"github.com/MichaelRBond/devdash/types"
)

type geocodingResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"results"`
}

type weatherResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		WeatherCode int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		Time    []string  `json:"time"`
		TempMax []float64 `json:"temperature_2m_max"`
		TempMin []float64 `json:"temperature_2m_min"`
		Code    []int     `json:"weather_code"`
	} `json:"daily"`
}

type WeatherProvider struct {
	lat    float64
	lon    float64
	units  string
	client *http.Client
}

func NewWeatherProvider(cfg config.WeatherConfig) (*WeatherProvider, error) {
	if cfg.Location == "" {
		return nil, fmt.Errorf("no location configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1",
		url.QueryEscape(cfg.Location))

	resp, err := client.Get(geoURL)
	if err != nil {
		return nil, fmt.Errorf("geocoding request: %w", err)
	}
	defer resp.Body.Close()

	var geoResp geocodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		return nil, fmt.Errorf("decoding geocoding response: %w", err)
	}
	if len(geoResp.Results) == 0 {
		return nil, fmt.Errorf("location %q not found", cfg.Location)
	}

	units := cfg.Units
	if units == "" {
		units = "fahrenheit"
	}

	return &WeatherProvider{
		lat:    geoResp.Results[0].Latitude,
		lon:    geoResp.Results[0].Longitude,
		units:  units,
		client: client,
	}, nil
}

func (p *WeatherProvider) Fetch(ctx context.Context) (types.Weather, error) {
	tempUnit := "fahrenheit"
	if p.units == "celsius" {
		tempUnit = "celsius"
	}

	u := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,weather_code&daily=temperature_2m_max,temperature_2m_min,weather_code&temperature_unit=%s&timezone=auto&forecast_days=3",
		p.lat, p.lon, tempUnit,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return types.Weather{}, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return types.Weather{}, fmt.Errorf("weather request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.Weather{}, fmt.Errorf("weather API returned %d", resp.StatusCode)
	}

	var wResp weatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&wResp); err != nil {
		return types.Weather{}, fmt.Errorf("decoding weather: %w", err)
	}

	weather := types.Weather{
		CurrentTemp: wResp.Current.Temperature,
		CurrentCode: wResp.Current.WeatherCode,
		CurrentDesc: weatherDesc(wResp.Current.WeatherCode),
		Available:   true,
	}

	now := time.Now()
	for i, dateStr := range wResp.Daily.Time {
		if i >= 3 {
			break
		}
		dayName := formatDayName(dateStr, now, i)
		weather.Forecast = append(weather.Forecast, types.DayForecast{
			Date: dayName,
			High: wResp.Daily.TempMax[i],
			Low:  wResp.Daily.TempMin[i],
			Code: wResp.Daily.Code[i],
		})
	}

	return weather, nil
}

func formatDayName(dateStr string, now time.Time, index int) string {
	switch index {
	case 0:
		return "Today"
	case 1:
		return "Tomorrow"
	default:
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return dateStr
		}
		return t.Format("Mon")
	}
}

func weatherIcon(code int) string {
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

func weatherDesc(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code >= 1 && code <= 3:
		return "Partly cloudy"
	case code == 45 || code == 48:
		return "Fog"
	case code >= 51 && code <= 57:
		return "Drizzle"
	case code >= 61 && code <= 67:
		return "Rain"
	case code >= 71 && code <= 77:
		return "Snow"
	case code >= 80 && code <= 82:
		return "Rain showers"
	case code >= 85 && code <= 86:
		return "Snow showers"
	case code >= 95 && code <= 99:
		return "Thunderstorm"
	default:
		return "Unknown"
	}
}

func (p *WeatherProvider) UnitSymbol() string {
	if p.units == "celsius" {
		return "°C"
	}
	return "°F"
}
