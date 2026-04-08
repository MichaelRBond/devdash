package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunSkill writes metadata to a temp JSON file and launches claude in a new zellij tab.
// For github and linear skills, it resolves the repo path and cds into it before running claude.
func RunSkill(skill Skill, metadata map[string]any) error {
	// Write metadata JSON to temp file.
	filename := fmt.Sprintf("/tmp/devdash-skill-%d.json", time.Now().UnixNano())
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing metadata file: %w", err)
	}

	// Resolve the working directory based on panel type.
	workDir := resolveWorkDir(metadata)

	// Build the shell command: cd to repo, then run claude.
	claudeCmd := fmt.Sprintf(
		"DEVDASH_CONTEXT=%s claude --append-system-prompt 'Context file available at %s — read it for metadata about the selected item.' 'Read %s and run /%s'",
		shellQuote(filename),
		filename,
		filename,
		skill.Name,
	)
	if workDir != "" {
		claudeCmd = fmt.Sprintf("cd %s && %s", shellQuote(workDir), claudeCmd)
	}

	// Launch in a new zellij tab.
	cmd := exec.Command("zellij", "action", "new-tab", "--", "sh", "-c", claudeCmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launching zellij tab: %w", err)
	}

	return nil
}

// resolveWorkDir attempts to find the local repo path for github/linear panels.
// Checks $GITHOME/<repo_name> and common locations.
func resolveWorkDir(metadata map[string]any) string {
	panel, _ := metadata["panel"].(string)
	if panel != "github" && panel != "linear" {
		return ""
	}

	repoName, _ := metadata["repo_name"].(string)
	if repoName == "" {
		return ""
	}

	// Check $GITHOME first.
	if gitHome := os.Getenv("GITHOME"); gitHome != "" {
		candidates := []string{
			filepath.Join(gitHome, repoName),
			filepath.Join(gitHome, repoName, repoName), // nested: $GITHOME/repo/repo
		}
		// Also check org/repo paths.
		fullRepo, _ := metadata["repo"].(string)
		if fullRepo != "" {
			candidates = append(candidates, filepath.Join(gitHome, fullRepo))
		}
		for _, c := range candidates {
			if isGitRepo(c) {
				return c
			}
		}
	}

	// Check common locations.
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	candidates := []string{
		filepath.Join(home, "Documents", "GIT", repoName),
		filepath.Join(home, "src", repoName),
		filepath.Join(home, "repos", repoName),
		filepath.Join(home, "code", repoName),
		filepath.Join(home, "projects", repoName),
		filepath.Join(home, repoName),
	}

	for _, c := range candidates {
		if isGitRepo(c) {
			return c
		}
	}

	return ""
}

// isGitRepo checks if a path is a git repository.
func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// findRepoPath is exported for skills that need to resolve repo paths.
func FindRepoPath(repoName, fullRepo string) string {
	return resolveWorkDir(map[string]any{
		"panel":     "github",
		"repo_name": repoName,
		"repo":      fullRepo,
	})
}
