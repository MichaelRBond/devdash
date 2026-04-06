package providers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelRBond/devdash/types"
)

func TestParseGitHubGraphQLResponse(t *testing.T) {
	raw := `{
		"data": {
			"reviewRequested": {
				"nodes": [
					{
						"repository": {"nameWithOwner": "org/repo"},
						"title": "Fix login bug",
						"number": 42,
						"author": {"login": "alice"},
						"url": "https://github.com/org/repo/pull/42",
						"createdAt": "2026-04-01T10:00:00Z",
						"updatedAt": "2026-04-05T12:00:00Z"
					}
				]
			},
			"authored": {
				"nodes": [
					{
						"repository": {"nameWithOwner": "org/repo2"},
						"title": "Add feature X",
						"number": 99,
						"author": {"login": "me"},
						"url": "https://github.com/org/repo2/pull/99",
						"createdAt": "2026-04-02T08:00:00Z",
						"updatedAt": "2026-04-06T09:00:00Z",
						"comments": {"totalCount": 3},
						"commits": {
							"nodes": [{
								"commit": {
									"statusCheckRollup": {
										"state": "SUCCESS"
									}
								}
							}]
						}
					}
				]
			}
		}
	}`

	var resp graphQLResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	reviewPRs, myPRs := resp.toPRs()

	if len(reviewPRs) != 1 {
		t.Fatalf("expected 1 review PR, got %d", len(reviewPRs))
	}
	pr := reviewPRs[0]
	if pr.Repo != "org/repo" {
		t.Errorf("expected repo org/repo, got %s", pr.Repo)
	}
	if pr.Title != "Fix login bug" {
		t.Errorf("expected title 'Fix login bug', got %s", pr.Title)
	}
	if pr.Number != 42 {
		t.Errorf("expected number 42, got %d", pr.Number)
	}
	if pr.Author != "alice" {
		t.Errorf("expected author alice, got %s", pr.Author)
	}

	if len(myPRs) != 1 {
		t.Fatalf("expected 1 authored PR, got %d", len(myPRs))
	}
	mine := myPRs[0]
	if mine.CommentCount != 3 {
		t.Errorf("expected 3 comments, got %d", mine.CommentCount)
	}
	if mine.CIStatus != types.CIStatusPassed {
		t.Errorf("expected CI passed, got %s", mine.CIStatus)
	}
}

func TestMapCIStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected types.CIStatus
	}{
		{"SUCCESS", types.CIStatusPassed},
		{"FAILURE", types.CIStatusFailed},
		{"PENDING", types.CIStatusPending},
		{"ERROR", types.CIStatusFailed},
		{"", types.CIStatusUnknown},
		{"EXPECTED", types.CIStatusUnknown},
	}

	for _, tc := range tests {
		got := mapCIStatus(tc.input)
		if got != tc.expected {
			t.Errorf("mapCIStatus(%q) = %s, want %s", tc.input, got, tc.expected)
		}
	}
}

func TestPRAge(t *testing.T) {
	now := time.Now()
	fourDaysAgo := now.Add(-4 * 24 * time.Hour)
	pr := types.PR{CreatedAt: fourDaysAgo}
	age := time.Since(pr.CreatedAt)
	if age < 3*24*time.Hour || age > 5*24*time.Hour {
		t.Errorf("unexpected age: %v", age)
	}
}
