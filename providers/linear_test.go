package providers

import (
	"encoding/json"
	"testing"
)

func TestParseLinearResponse(t *testing.T) {
	raw := `{
		"data": {
			"issues": {
				"nodes": [
					{
						"identifier": "PE-2823",
						"title": "Fix dashboard layout",
						"url": "https://linear.app/team/issue/PE-2823",
						"createdAt": "2026-04-01T10:00:00Z",
						"state": {"name": "In Progress", "type": "started"},
						"team": {"key": "PE", "name": "Platform Engineering"},
						"project": {"name": "Dashboard Revamp"},
						"labels": {"nodes": [{"name": "Frontend"}, {"name": "Bug"}]}
					},
					{
						"identifier": "PE-2824",
						"title": "Add unit tests",
						"url": "https://linear.app/team/issue/PE-2824",
						"createdAt": "2026-04-02T08:00:00Z",
						"state": {"name": "Todo", "type": "unstarted"},
						"team": {"key": "PE", "name": "Platform Engineering"},
						"project": null,
						"labels": {"nodes": []}
					}
				]
			}
		}
	}`

	var resp linearResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stateGroups := map[string]string{
		"In Progress": "in_progress",
		"Todo":        "up_next",
	}
	tasks := resp.toTasks(stateGroups)

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Identifier != "PE-2823" {
		t.Errorf("expected PE-2823, got %s", task.Identifier)
	}
	if task.Team != "Platform Engineering" {
		t.Errorf("expected team Platform Engineering, got %s", task.Team)
	}
	if task.StateGroup != "in_progress" {
		t.Errorf("expected state group in_progress, got %s", task.StateGroup)
	}
	if len(task.Labels) != 2 || task.Labels[0] != "Frontend" {
		t.Errorf("unexpected labels: %v", task.Labels)
	}
	if task.Project != "Dashboard Revamp" {
		t.Errorf("expected project Dashboard Revamp, got %s", task.Project)
	}

	task2 := tasks[1]
	if task2.StateGroup != "up_next" {
		t.Errorf("expected state group up_next, got %s", task2.StateGroup)
	}
	if task2.Project != "" {
		t.Errorf("expected empty project, got %s", task2.Project)
	}
}

func TestLinearStateGroupMapping(t *testing.T) {
	stateGroups := map[string]string{
		"In Progress": "in_progress",
		"Up Next":     "up_next",
		"In Review":   "in_review",
	}

	tests := []struct {
		state    string
		expected string
	}{
		{"In Progress", "in_progress"},
		{"Up Next", "up_next"},
		{"In Review", "in_review"},
		{"Unknown State", ""},
	}

	for _, tc := range tests {
		got := stateGroups[tc.state]
		if got != tc.expected {
			t.Errorf("state %q: expected group %q, got %q", tc.state, tc.expected, got)
		}
	}
}
