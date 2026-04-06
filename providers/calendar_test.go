package providers

import (
	"testing"
	"time"

	"github.com/MichaelRBond/devdash/types"
)

func TestMapEventStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected types.EventStatus
	}{
		{"accepted", types.EventAccepted},
		{"tentative", types.EventTentative},
		{"declined", types.EventDeclined},
		{"needsAction", types.EventTentative},
		{"", types.EventTentative},
	}

	for _, tc := range tests {
		got := mapEventStatus(tc.input)
		if got != tc.expected {
			t.Errorf("mapEventStatus(%q) = %s, want %s", tc.input, got, tc.expected)
		}
	}
}

func TestCalendarEventToType(t *testing.T) {
	start := time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 7, 15, 0, 0, 0, time.UTC)

	event := calendarEventItem{
		Summary:  "Team Standup",
		Start:    eventTime{DateTime: start.Format(time.RFC3339)},
		End:      eventTime{DateTime: end.Format(time.RFC3339)},
		Location: "Zoom",
		HTMLURL:  "https://calendar.google.com/event/123",
		Attendees: []eventAttendee{
			{Self: true, ResponseStatus: "accepted"},
		},
	}

	result := event.toEvent()
	if result.Title != "Team Standup" {
		t.Errorf("expected title 'Team Standup', got %s", result.Title)
	}
	if result.Status != types.EventAccepted {
		t.Errorf("expected accepted, got %s", result.Status)
	}
	if result.Location != "Zoom" {
		t.Errorf("expected location Zoom, got %s", result.Location)
	}
}
