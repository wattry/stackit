// Package track implements the stackit track command for tracking branches in a stack.
package track

import (
	"context"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/output"
)

// Handler receives events from track action
type Handler interface {
	// PromptSelectParent prompts user to select a parent for the branch
	// Returns the selected parent branch name
	PromptSelectParent(ctx context.Context, eng engine.Engine, ghClient github.Client, logger output.Logger, branchName string) (string, error)

	// PromptTrackChild prompts user to confirm tracking a child branch
	// Returns true to track the child, false to skip
	PromptTrackChild(childName, parentName string) (bool, error)

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool
}

// NullHandler is a no-op handler for when nil is passed
type NullHandler struct{}

// PromptSelectParent implements Handler. Returns empty string for null handler.
func (h *NullHandler) PromptSelectParent(_ context.Context, _ engine.Engine, _ github.Client, _ output.Logger, _ string) (string, error) {
	return "", nil
}

// PromptTrackChild implements Handler. Returns false for null handler.
func (h *NullHandler) PromptTrackChild(_, _ string) (bool, error) { return false, nil }

// Cleanup implements Handler.
func (h *NullHandler) Cleanup() {}

// IsInteractive implements Handler.
func (h *NullHandler) IsInteractive() bool { return false }
