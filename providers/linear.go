package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/MichaelRBond/devdash/config"
	"github.com/MichaelRBond/devdash/types"
)

type linearResponse struct {
	Data struct {
		Issues struct {
			Nodes []linearIssueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"data"`
}

type linearIssueNode struct {
	Identifier string    `json:"identifier"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"createdAt"`
	State      struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Team struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"team"`
	Project *struct {
		Name string `json:"name"`
	} `json:"project"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
}

func (r *linearResponse) toTasks(stateGroups map[string]string) []types.Task {
	var tasks []types.Task
	for _, n := range r.Data.Issues.Nodes {
		var labels []string
		for _, l := range n.Labels.Nodes {
			labels = append(labels, l.Name)
		}

		project := ""
		if n.Project != nil {
			project = n.Project.Name
		}

		tasks = append(tasks, types.Task{
			Identifier: n.Identifier,
			Title:      n.Title,
			Team:       n.Team.Name,
			Project:    project,
			URL:        n.URL,
			CreatedAt:  n.CreatedAt,
			Labels:     labels,
			State:      n.State.Name,
			StateGroup: stateGroups[n.State.Name],
		})
	}
	return tasks
}

const linearQuery = `query($teamKeys: [String!]) {
  issues(
    filter: {
      assignee: { isMe: { eq: true } }
      state: { type: { nin: ["completed", "canceled"] } }
      team: { key: { in: $teamKeys } }
    }
    first: 100
    orderBy: updatedAt
  ) {
    nodes {
      identifier
      title
      url
      createdAt
      state { name type }
      team { key name }
      project { name }
      labels { nodes { name } }
    }
  }
}`

// LinearProvider fetches tasks from the Linear GraphQL API.
type LinearProvider struct {
	apiKey      string
	teamKeys    []string
	stateGroups map[string]string
	client      *http.Client
}

// NewLinearProvider creates a Linear provider from config.
func NewLinearProvider(cfg config.LinearConfig) (*LinearProvider, error) {
	key := os.Getenv(cfg.TokenEnv)
	if key == "" {
		return nil, fmt.Errorf("env var %s not set", cfg.TokenEnv)
	}

	stateGroups := make(map[string]string)
	for _, s := range cfg.StatesUpNext {
		stateGroups[s] = "up_next"
	}
	for _, s := range cfg.StatesInProgress {
		stateGroups[s] = "in_progress"
	}
	for _, s := range cfg.StatesInReview {
		stateGroups[s] = "in_review"
	}

	return &LinearProvider{
		apiKey:      key,
		teamKeys:    cfg.TeamKeys,
		stateGroups: stateGroups,
		client:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Fetch returns tasks assigned to the authenticated user.
func (p *LinearProvider) Fetch(ctx context.Context) ([]types.Task, error) {
	body := map[string]any{
		"query": linearQuery,
		"variables": map[string]any{
			"teamKeys": p.teamKeys,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.linear.app/graphql", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Linear API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Linear API returned %d", resp.StatusCode)
	}

	var linResp linearResponse
	if err := json.NewDecoder(resp.Body).Decode(&linResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return linResp.toTasks(p.stateGroups), nil
}
