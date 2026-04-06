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
	TeamReview   bool // true if assigned via team, false if directly assigned
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

// Usage represents Claude Code usage data.
type Usage struct {
	MessagesUsed    int
	MessageLimit    int
	TokensIn        int64
	TokensOut       int64
	CacheCreateIn   int64
	CacheReadIn     int64
	TotalTokens     int64
	CostUSD         float64
	ResetAt         time.Time
	Model           string
	Plan            string
	Available       bool
	ModelStats      []ModelUsage
	BurnRate        float64   // tokens per minute
	TokenLimit      int64     // plan token limit per window
	WindowStart     time.Time // earliest usage in the rolling window
	ActiveMinutes   float64   // minutes of active usage in window
}

// ModelUsage holds per-model token and cost breakdown.
type ModelUsage struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
	CacheCreate  int64
	CacheRead    int64
	TotalTokens  int64
	CostUSD      float64
}

// Provider is the interface all data sources implement.
type Provider[T any] interface {
	Fetch(ctx context.Context) ([]T, error)
}
