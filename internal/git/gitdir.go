package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetGitDir resolves the actual git directory for a repository.
// In worktrees, .git is a file pointing to the real git directory, so we need
// to use git rev-parse to get the correct path.
//
// This returns the worktree-specific git directory (use GetGitCommonDir for shared config).
func GetGitDir(repoRoot string) string {
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

// getGitDir is the runner method version that uses RunGitCommandWithContext.
// Returns the worktree-specific git directory.
func (r *runner) getGitDir(ctx context.Context) string {
	output, err := r.RunGitCommandWithContext(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		// Fallback to non-absolute if absolute-git-dir fails (older git versions)
		output, err = r.RunGitCommandWithContext(ctx, "rev-parse", "--git-dir")
		if err != nil {
			// Final fallback: assume standard .git directory
			return filepath.Join(r.repoRoot, ".git")
		}
	}

	gitDir := strings.TrimSpace(output)
	// If gitDir is relative and we have repoRoot, make it absolute
	if !filepath.IsAbs(gitDir) && r.repoRoot != "" {
		gitDir = filepath.Join(r.repoRoot, gitDir)
	}
	return gitDir
}
