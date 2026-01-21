package lock

import (
	"stackit.dev/stackit/internal/actions/submit"
)

// Handler receives events from lock/unlock actions
type Handler interface {
	// PromptSubmitBeforeLock prompts user to submit unpushed changes before locking
	// Returns true to submit before locking, false to skip
	PromptSubmitBeforeLock(unpushedBranches []string) (bool, error)

	// PromptUnlockDownstack prompts user to also unlock downstack locked branches
	// Returns true to unlock downstack branches, false to skip
	PromptUnlockDownstack(lockedBranchNames []string) (bool, error)

	// GetSubmitHandler returns a handler for the nested submit operation
	GetSubmitHandler() submit.Handler

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool
}

// NullHandler is a no-op handler for when nil is passed
type NullHandler struct{}

// PromptSubmitBeforeLock implements Handler. Returns false (skip submit) for null handler.
func (h *NullHandler) PromptSubmitBeforeLock(_ []string) (bool, error) { return false, nil }

// PromptUnlockDownstack implements Handler. Returns false (skip) for null handler.
func (h *NullHandler) PromptUnlockDownstack(_ []string) (bool, error) { return false, nil }

// GetSubmitHandler implements Handler. Returns nil for null handler.
func (h *NullHandler) GetSubmitHandler() submit.Handler { return nil }

// Cleanup implements Handler.
func (h *NullHandler) Cleanup() {}

// IsInteractive implements Handler.
func (h *NullHandler) IsInteractive() bool { return false }
