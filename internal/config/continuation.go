package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ContinuationState represents the state of a command that was interrupted by a rebase conflict
type ContinuationState struct {
	BranchesToRestack     []string `json:"branchesToRestack,omitempty"`
	BranchesToSync        []string `json:"branchesToSync,omitempty"` // For future sync command
	CurrentBranchOverride string   `json:"currentBranchOverride,omitempty"`
	RebasedBranchBase     string   `json:"rebasedBranchBase,omitempty"`
}

// getGitDir resolves the actual git directory for a repository.
// In worktrees, .git is a file pointing to the real git directory, so we need
// to use git rev-parse to get the correct path.
func getGitDir(repoRoot string) string {
	// Try --absolute-git-dir first (git 2.13+), then fall back to --git-dir
	cmd := exec.Command("git", "rev-parse", "--absolute-git-dir")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		// Fallback to --git-dir for older git versions
		cmd = exec.Command("git", "rev-parse", "--git-dir")
		cmd.Dir = repoRoot
		output, err = cmd.Output()
		if err != nil {
			// Final fallback: assume standard .git directory
			return filepath.Join(repoRoot, ".git")
		}
	}

	gitDir := strings.TrimSpace(string(output))
	// If gitDir is relative, make it absolute
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoRoot, gitDir)
	}
	return gitDir
}

// GetContinuationState reads the continuation state from disk
func GetContinuationState(repoRoot string) (*ContinuationState, error) {
	gitDir := getGitDir(repoRoot)
	configPath := filepath.Join(gitDir, ".stackit_continue")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no continuation state found")
		}
		return nil, fmt.Errorf("failed to read continuation state: %w", err)
	}

	var state ContinuationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse continuation state: %w", err)
	}
	return &state, nil
}

// PersistContinuationState writes the continuation state to disk
func PersistContinuationState(repoRoot string, state *ContinuationState) error {
	gitDir := getGitDir(repoRoot)
	configPath := filepath.Join(gitDir, ".stackit_continue")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal continuation state: %w", err)
	}
	return os.WriteFile(configPath, data, 0600)
}

// ClearContinuationState removes the continuation state file
func ClearContinuationState(repoRoot string) error {
	gitDir := getGitDir(repoRoot)
	configPath := filepath.Join(gitDir, ".stackit_continue")
	err := os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear continuation state: %w", err)
	}
	return nil
}
