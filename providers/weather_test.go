package providers

import (
	"encoding/json"
	"testing"
)

func TestParseGeocodingResponse(t *testing.T) {
	raw := `{
		"results": [
			{
				"name": "Austin",
				"latitude": 30.2672,
				"longitude": -97.7431,
				"country": "United States"
			}
		]
	}`

	var resp geocodingResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Latitude != 30.2672 {
		t.Errorf("expected lat 30.2672, got %f", resp.Results[0].Latitude)
	}
	if resp.Results[0].Longitude != -97.7431 {
		t.Errorf("expected lon -97.7431, got %f", resp.Results[0].Longitude)
	}
}

func TestParseWeatherResponse(t *testing.T) {
	raw := `{
		"current": {
			"temperature_2m": 72.5,
			"weather_code": 0
		},
		"daily": {
			"time": ["2026-04-09", "2026-04-10", "2026-04-11"],
			"temperature_2m_max": [75.0, 68.0, 70.0],
			"temperature_2m_min": [58.0, 55.0, 56.0],
			"weather_code": [0, 61, 2]
		}
	}`

	var resp weatherResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Current.Temperature != 72.5 {
		t.Errorf("expected temp 72.5, got %f", resp.Current.Temperature)
	}
	if resp.Current.WeatherCode != 0 {
		t.Errorf("expected code 0, got %d", resp.Current.WeatherCode)
	}
	if len(resp.Daily.Time) != 3 {
		t.Fatalf("expected 3 days, got %d", len(resp.Daily.Time))
	}
	if resp.Daily.TempMax[0] != 75.0 {
		t.Errorf("expected max 75.0, got %f", resp.Daily.TempMax[0])
	}
}

func TestWeatherCodeToIcon(t *testing.T) {
	tests := []struct {
		code int
		icon string
	}{
		{0, "☀"},
		{2, "⛅"},
		{45, "🌫"},
		{61, "🌧"},
		{71, "❄"},
		{95, "⛈"},
		{999, "?"},
	}
	for _, tc := range tests {
		got := weatherIcon(tc.code)
		if got != tc.icon {
			t.Errorf("weatherIcon(%d) = %s, want %s", tc.code, got, tc.icon)
		}
	}
}

func TestWeatherCodeToDesc(t *testing.T) {
	tests := []struct {
		code int
		desc string
	}{
		{0, "Clear"},
		{2, "Partly cloudy"},
		{61, "Rain"},
		{71, "Snow"},
		{95, "Thunderstorm"},
	}
	for _, tc := range tests {
		got := weatherDesc(tc.code)
		if got != tc.desc {
			t.Errorf("weatherDesc(%d) = %s, want %s", tc.code, got, tc.desc)
		}
	}
}
