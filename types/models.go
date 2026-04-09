package types

import (
	"context"
	"time"
)

// PR represents a GitHub pull request.
type PR struct {
	Repo         string
	Title        string
	Number       int
	Author       string
	URL          string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CommentCount int
	CIStatus     CIStatus
	ReviewStatus ReviewStatus
	TeamReview   bool
	Branch       string
	IsDraft      bool
}

type CIStatus string

const (
	CIStatusPending CIStatus = "pending"
	CIStatusPassed  CIStatus = "passed"
	CIStatusFailed  CIStatus = "failed"
	CIStatusUnknown CIStatus = "unknown"
)

type ReviewStatus string

const (
	ReviewPending  ReviewStatus = "pending"
	ReviewApproved ReviewStatus = "approved"
	ReviewChanges  ReviewStatus = "changes_requested"
)

// Task represents a Linear issue.
type Task struct {
	Identifier string
	Title      string
	Team       string
	Project    string
	URL        string
	CreatedAt  time.Time
	Labels     []string
	State      string
	StateGroup string // "up_next", "in_progress", "in_review"
}

// Event represents a calendar event.
type Event struct {
	Title      string
	StartTime  time.Time
	EndTime    time.Time
	Status     EventStatus
	Location   string
	URL        string
	MeetingURL string // Google Meet, Zoom, etc.
}

type EventStatus string

const (
	EventAccepted  EventStatus = "accepted"
	EventTentative EventStatus = "tentative"
	EventDeclined  EventStatus = "declined"
)

// Weather holds current conditions and forecast.
type Weather struct {
	CurrentTemp float64
	CurrentCode int
	CurrentDesc string
	Forecast    []DayForecast
	Available   bool
}

// DayForecast holds one day's forecast data.
type DayForecast struct {
	Date string
	High float64
	Low  float64
	Code int
}

// Provider is the interface all data sources implement.
type Provider[T any] interface {
	Fetch(ctx context.Context) ([]T, error)
}
