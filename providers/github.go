package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/MichaelRBond/devdash/config"
	"github.com/MichaelRBond/devdash/types"
)

// graphQLResponse maps the GitHub GraphQL response for our PR query.
type graphQLResponse struct {
	Data struct {
		ReviewRequested struct {
			Nodes []ghPRNode `json:"nodes"`
		} `json:"reviewRequested"`
		Authored struct {
			Nodes []ghPRNode `json:"nodes"`
		} `json:"authored"`
	} `json:"data"`
}

type ghPRNode struct {
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Title     string                 `json:"title"`
	Number    int                    `json:"number"`
	Author    struct{ Login string } `json:"author"`
	URL       string                 `json:"url"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Comments  struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

func (r *graphQLResponse) toPRs() (reviewPRs []types.PR, myPRs []types.PR) {
	for _, n := range r.Data.ReviewRequested.Nodes {
		reviewPRs = append(reviewPRs, n.toPR())
	}
	for _, n := range r.Data.Authored.Nodes {
		myPRs = append(myPRs, n.toPR())
	}

	// Review PRs: oldest first (most urgent).
	sort.Slice(reviewPRs, func(i, j int) bool {
		return reviewPRs[i].CreatedAt.Before(reviewPRs[j].CreatedAt)
	})
	// My PRs: most recently updated first.
	sort.Slice(myPRs, func(i, j int) bool {
		return myPRs[i].UpdatedAt.After(myPRs[j].UpdatedAt)
	})

	return reviewPRs, myPRs
}

func (n ghPRNode) toPR() types.PR {
	ci := types.CIStatusUnknown
	if len(n.Commits.Nodes) > 0 && n.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
		ci = mapCIStatus(n.Commits.Nodes[0].Commit.StatusCheckRollup.State)
	}

	return types.PR{
		Repo:         n.Repository.NameWithOwner,
		Title:        n.Title,
		Number:       n.Number,
		Author:       n.Author.Login,
		URL:          n.URL,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
		CommentCount: n.Comments.TotalCount,
		CIStatus:     ci,
	}
}

func mapCIStatus(state string) types.CIStatus {
	switch state {
	case "SUCCESS":
		return types.CIStatusPassed
	case "FAILURE", "ERROR":
		return types.CIStatusFailed
	case "PENDING":
		return types.CIStatusPending
	default:
		return types.CIStatusUnknown
	}
}

const prQuery = `query($reviewQuery: String!, $authorQuery: String!) {
  reviewRequested: search(query: $reviewQuery, type: ISSUE, first: 50) {
    nodes {
      ... on PullRequest {
        repository { nameWithOwner }
        title
        number
        author { login }
        url
        createdAt
        updatedAt
      }
    }
  }
  authored: search(query: $authorQuery, type: ISSUE, first: 50) {
    nodes {
      ... on PullRequest {
        repository { nameWithOwner }
        title
        number
        author { login }
        url
        createdAt
        updatedAt
        comments { totalCount }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup { state }
            }
          }
        }
      }
    }
  }
}`

// GitHubProvider fetches PRs from GitHub's GraphQL API.
type GitHubProvider struct {
	token          string
	username       string
	orgs           []string
	repos          []string
	teamSlugs      []string
	client         *http.Client
}

// NewGitHubProvider creates a GitHub provider from config.
func NewGitHubProvider(cfg config.GitHubConfig) (*GitHubProvider, error) {
	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return nil, fmt.Errorf("env var %s not set", cfg.TokenEnv)
	}
	p := &GitHubProvider{
		token:     token,
		orgs:      cfg.Orgs,
		repos:     cfg.Repos,
		teamSlugs: cfg.ReviewTeamSlugs,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
	// Resolve the authenticated user's login — @me doesn't work in GraphQL search.
	username, err := p.fetchViewerLogin()
	if err != nil {
		return nil, fmt.Errorf("resolving GitHub username: %w", err)
	}
	p.username = username
	return p, nil
}

// fetchViewerLogin queries the GitHub API for the authenticated user's login.
func (p *GitHubProvider) fetchViewerLogin() (string, error) {
	body := map[string]any{"query": `query { viewer { login } }`}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Viewer struct {
				Login string `json:"login"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Data.Viewer.Login == "" {
		return "", fmt.Errorf("empty login returned — check token scopes")
	}
	return result.Data.Viewer.Login, nil
}

// FetchAll returns (reviewPRs, myPRs) from a single GraphQL query.
func (p *GitHubProvider) FetchAll(ctx context.Context) ([]types.PR, []types.PR, error) {
	orgFilter := ""
	for _, org := range p.orgs {
		orgFilter += fmt.Sprintf(" org:%s", org)
	}
	for _, repo := range p.repos {
		orgFilter += fmt.Sprintf(" repo:%s", repo)
	}

	reviewQuery := fmt.Sprintf("is:pr is:open review-requested:%s%s", p.username, orgFilter)
	authorQuery := fmt.Sprintf("is:pr is:open author:%s%s", p.username, orgFilter)

	gqlResp, err := p.executeQuery(ctx, reviewQuery, authorQuery)
	if err != nil {
		return nil, nil, err
	}

	review, mine := gqlResp.toPRs()

	// Also fetch team review requests and merge (deduplicated).
	for _, slug := range p.teamSlugs {
		for _, org := range p.orgs {
			teamQuery := fmt.Sprintf("is:pr is:open team-review-requested:%s/%s%s", org, slug, orgFilter)
			teamResp, err := p.executeQuery(ctx, teamQuery, "")
			if err != nil {
				continue
			}
			teamPRs, _ := teamResp.toPRs()
			for i := range teamPRs {
				teamPRs[i].TeamReview = true
			}
			review = mergeUniquePRs(review, teamPRs)
		}
	}

	// Sort: direct reviews first (oldest first), then team reviews (newest first).
	sort.SliceStable(review, func(i, j int) bool {
		if review[i].TeamReview != review[j].TeamReview {
			return !review[i].TeamReview // direct (false) before team (true)
		}
		if !review[i].TeamReview {
			return review[i].CreatedAt.Before(review[j].CreatedAt) // direct: oldest first
		}
		return review[i].CreatedAt.After(review[j].CreatedAt) // team: newest first
	})

	return review, mine, nil
}

// executeQuery runs a single GraphQL PR search query.
func (p *GitHubProvider) executeQuery(ctx context.Context, reviewQuery, authorQuery string) (*graphQLResponse, error) {
	if authorQuery == "" {
		// Use a query that returns nothing for the authored side.
		authorQuery = "is:pr is:open author:__none__"
	}

	body := map[string]any{
		"query": prQuery,
		"variables": map[string]string{
			"reviewQuery": reviewQuery,
			"authorQuery": authorQuery,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var gqlResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &gqlResp, nil
}

// mergeUniquePRs adds PRs from src into dst, skipping duplicates by PR number + repo.
func mergeUniquePRs(dst, src []types.PR) []types.PR {
	seen := make(map[string]bool)
	for _, pr := range dst {
		seen[fmt.Sprintf("%s#%d", pr.Repo, pr.Number)] = true
	}
	for _, pr := range src {
		key := fmt.Sprintf("%s#%d", pr.Repo, pr.Number)
		if !seen[key] {
			dst = append(dst, pr)
			seen[key] = true
		}
	}
	return dst
}
