package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stackit.dev/stackit/internal/app"
)

const precommitHookTemplate = `#!/bin/bash
# Installed by Stackit. To bypass, use --no-verify.
stackit precommit verify
`

// PrecommitInstallAction installs the pre-commit hook
func PrecommitInstallAction(ctx *app.Context) error {
	repoRoot := ctx.RepoRoot
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		content, err := os.ReadFile(hookPath)
		if err == nil && strings.Contains(string(content), "stackit precommit verify") {
			ctx.Splog.Info("Pre-commit hook is already installed.")
			return nil
		}

		// If it exists but doesn't have our command, we should probably append or warn.
		// For simplicity, let's append.
		f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return fmt.Errorf("failed to open existing pre-commit hook: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString("\n# Added by Stackit\nstackit precommit verify\n"); err != nil {
			return fmt.Errorf("failed to append to pre-commit hook: %w", err)
		}
		ctx.Splog.Info("Appended Stackit verification to existing pre-commit hook.")
	} else {
		// Create new hook
		if err := os.WriteFile(hookPath, []byte(precommitHookTemplate), 0755); err != nil {
			return fmt.Errorf("failed to write pre-commit hook: %w", err)
		}
		ctx.Splog.Info("Installed Stackit pre-commit hook.")
	}

	return nil
}

// PrecommitVerifyAction checks if the current branch can be modified
func PrecommitVerifyAction(ctx *app.Context) error {
	eng := ctx.Engine
	currentBranch := eng.CurrentBranch()

	if currentBranch == nil {
		// Not on a branch, allow the commit (e.g. initial commit or detached HEAD)
		return nil
	}

	// EnsureCanModify will return a BranchModificationError if locked or frozen
	return currentBranch.EnsureCanModify()
}
