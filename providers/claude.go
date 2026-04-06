package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MichaelRBond/devdash/config"
	"github.com/MichaelRBond/devdash/types"
)

const rollingWindow = 5 * time.Hour

// Pricing per million tokens by model family.
type modelPricing struct {
	Input       float64
	Output      float64
	CacheCreate float64
	CacheRead   float64
}

var pricing = map[string]modelPricing{
	"opus":   {Input: 15.0, Output: 75.0, CacheCreate: 18.75, CacheRead: 1.50},
	"sonnet": {Input: 3.0, Output: 15.0, CacheCreate: 3.75, CacheRead: 0.30},
	"haiku":  {Input: 0.25, Output: 1.25, CacheCreate: 0.30, CacheRead: 0.03},
}


// ClaudeProvider fetches Claude Code usage data from local session files.
type ClaudeProvider struct {
	plan       string
	tokenLimit int64
	claudeDir  string
}

func NewClaudeProvider(cfg config.ClaudeConfig) *ClaudeProvider {
	home, _ := os.UserHomeDir()
	return &ClaudeProvider{
		plan:       cfg.Plan,
		tokenLimit: cfg.TokenLimit,
		claudeDir:  filepath.Join(home, ".claude"),
	}
}

type sessionInfo struct {
	SessionID string `json:"sessionId"`
	StartedAt int64  `json:"startedAt"`
}

type sessionMessage struct {
	Type    string         `json:"type"`
	Role    string         `json:"role"`
	Model   string         `json:"model"`
	Usage   *sessionUsage  `json:"usage,omitempty"`
	Message *nestedMessage `json:"message,omitempty"`
}

type nestedMessage struct {
	Model string        `json:"model"`
	Usage *sessionUsage `json:"usage,omitempty"`
}

type sessionUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CacheRead    int64 `json:"cache_read_input_tokens"`
	CacheCreate  int64 `json:"cache_creation_input_tokens"`
}

// usageEntry is a single parsed usage record with model info.
type usageEntry struct {
	model       string
	inputTokens int64
	outputTokens int64
	cacheCreate int64
	cacheRead   int64
	timestamp   time.Time // file mod time as proxy
}

func (p *ClaudeProvider) Fetch(ctx context.Context) ([]types.Usage, error) {
	usage := types.Usage{
		Plan:      p.plan,
		Available: false,
	}

	now := time.Now()
	windowStart := now.Add(-rollingWindow)

	sessionStarts := p.loadSessionStarts(windowStart)
	entries, latestModel, oldest := p.collectEntries(windowStart, sessionStarts)

	if len(entries) == 0 && latestModel == "" {
		return []types.Usage{usage}, nil
	}

	// Aggregate totals and per-model stats.
	modelMap := make(map[string]*types.ModelUsage)
	var totalIn, totalOut, totalCacheCreate, totalCacheRead int64

	for _, e := range entries {
		totalIn += e.inputTokens
		totalOut += e.outputTokens
		totalCacheCreate += e.cacheCreate
		totalCacheRead += e.cacheRead

		mu, ok := modelMap[e.model]
		if !ok {
			mu = &types.ModelUsage{Model: e.model}
			modelMap[e.model] = mu
		}
		mu.InputTokens += e.inputTokens
		mu.OutputTokens += e.outputTokens
		mu.CacheCreate += e.cacheCreate
		mu.CacheRead += e.cacheRead
	}

	// Calculate per-model costs and totals.
	// TotalTokens = input + output only (rate-limit metric).
	// Cost uses all 4 types (billing metric).
	var modelStats []types.ModelUsage
	var totalCost float64
	for _, mu := range modelMap {
		mu.TotalTokens = mu.InputTokens + mu.OutputTokens
		mu.CostUSD = calcCost(mu.Model, mu.InputTokens, mu.OutputTokens, mu.CacheCreate, mu.CacheRead)
		totalCost += mu.CostUSD
		modelStats = append(modelStats, *mu)
	}
	sort.Slice(modelStats, func(i, j int) bool {
		return modelStats[i].TotalTokens > modelStats[j].TotalTokens
	})

	// Rate-limit tokens = input + output (cache tokens don't count toward limits).
	total := totalIn + totalOut

	// Calculate burn rate (tokens per minute of active time).
	activeMinutes := now.Sub(oldest).Minutes()
	if activeMinutes < 1 {
		activeMinutes = 1
	}
	burnRate := float64(total) / activeMinutes

	usage.TokensIn = totalIn
	usage.TokensOut = totalOut
	usage.CacheCreateIn = totalCacheCreate
	usage.CacheReadIn = totalCacheRead
	usage.TotalTokens = total
	usage.CostUSD = totalCost
	usage.Model = latestModel
	usage.ModelStats = modelStats
	usage.BurnRate = burnRate
	usage.ActiveMinutes = activeMinutes
	usage.TokenLimit = p.tokenLimit
	usage.Available = true


	if !oldest.IsZero() {
		usage.WindowStart = oldest
		usage.ResetAt = oldest.Add(rollingWindow)
	}

	return []types.Usage{usage}, nil
}

func (p *ClaudeProvider) loadSessionStarts(windowStart time.Time) map[string]time.Time {
	result := make(map[string]time.Time)
	sessDir := filepath.Join(p.claudeDir, "sessions")
	entries, err := os.ReadDir(sessDir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sessDir, entry.Name()))
		if err != nil {
			continue
		}
		var info sessionInfo
		if err := json.Unmarshal(data, &info); err != nil || info.SessionID == "" || info.StartedAt == 0 {
			continue
		}
		startTime := time.UnixMilli(info.StartedAt)
		if startTime.Before(windowStart) {
			continue
		}
		result[info.SessionID] = startTime
	}
	return result
}

// collectEntries scans JSONL files and returns all usage entries in the rolling window.
func (p *ClaudeProvider) collectEntries(windowStart time.Time, sessionStarts map[string]time.Time) ([]usageEntry, string, time.Time) {
	projectsDir := filepath.Join(p.claudeDir, "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		return nil, "", time.Time{}
	}

	type fileInfo struct {
		path      string
		sessionID string
		modTime   time.Time
	}
	var files []fileInfo

	filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.ModTime().Before(windowStart) {
			return nil
		}
		base := filepath.Base(path)
		sessionID := strings.TrimSuffix(base, ".jsonl")
		files = append(files, fileInfo{path: path, sessionID: sessionID, modTime: info.ModTime()})
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	var allEntries []usageEntry
	var latestModel string
	var oldest time.Time

	for _, fi := range files {
		entries, mdl := scanSessionEntries(fi.path)
		if len(entries) == 0 {
			continue
		}

		allEntries = append(allEntries, entries...)

		if latestModel == "" && mdl != "" {
			latestModel = mdl
		}

		if startTime, ok := sessionStarts[fi.sessionID]; ok {
			if oldest.IsZero() || startTime.Before(oldest) {
				oldest = startTime
			}
		}
	}

	return allEntries, latestModel, oldest
}

// scanSessionEntries reads a JSONL file and returns individual usage entries.
func scanSessionEntries(path string) ([]usageEntry, string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var entries []usageEntry
	var model string

	for scanner.Scan() {
		var msg sessionMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		u := msg.Usage
		if u == nil && msg.Message != nil {
			u = msg.Message.Usage
		}
		if u == nil {
			continue
		}

		mdl := msg.Model
		if mdl == "" && msg.Message != nil {
			mdl = msg.Message.Model
		}
		if mdl != "" {
			model = mdl
		}

		entries = append(entries, usageEntry{
			model:        mdl,
			inputTokens:  u.InputTokens,
			outputTokens: u.OutputTokens,
			cacheCreate:  u.CacheCreate,
			cacheRead:    u.CacheRead,
		})
	}

	return entries, model
}

// calcCost computes the USD cost for a given model and token counts.
func calcCost(model string, input, output, cacheCreate, cacheRead int64) float64 {
	family := modelFamily(model)
	p, ok := pricing[family]
	if !ok {
		p = pricing["sonnet"] // default fallback
	}
	return float64(input)/1_000_000*p.Input +
		float64(output)/1_000_000*p.Output +
		float64(cacheCreate)/1_000_000*p.CacheCreate +
		float64(cacheRead)/1_000_000*p.CacheRead
}

// modelFamily extracts "opus", "sonnet", or "haiku" from a model name.
func modelFamily(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "opus"):
		return "opus"
	case strings.Contains(lower, "haiku"):
		return "haiku"
	default:
		return "sonnet"
	}
}
