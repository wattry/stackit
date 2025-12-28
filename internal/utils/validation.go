package utils

import (
	"context"
	"fmt"
	"os"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// IsInteractive checks if we're in an interactive terminal
func IsInteractive() bool {
	return tui.IsTTY()
}

// ValidateOnBranch ensures the user is on a branch
func ValidateOnBranch(engine engine.Engine) (string, error) {
	currentBranch := engine.CurrentBranch()
	if currentBranch == nil {
		return "", fmt.Errorf("not on a branch")
	}
	return currentBranch.GetName(), nil
}

// CheckRebaseInProgress ensures no rebase is currently active
func CheckRebaseInProgress(ctx context.Context) error {
	if git.IsRebaseInProgress(ctx) {
		return fmt.Errorf("a rebase is already in progress. Please finish or abort it first")
	}
	return nil
}

// HasUncommittedChanges checks if there are uncommitted changes in the repository
func HasUncommittedChanges(ctx context.Context) bool {
	output, err := git.RunGitCommandWithContext(ctx, "status", "--porcelain")
	if err != nil {
		return false
	}
	return output != ""
}

// IsDemoMode returns true if STACKIT_DEMO environment variable is set
func IsDemoMode() bool {
	return os.Getenv("STACKIT_DEMO") != ""
}
