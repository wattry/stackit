// Package untrack implements the stackit untrack command for stopping branch tracking.
package untrack

// Handler receives events from untrack action
type Handler interface {
	// PromptConfirmUntrackDescendants prompts user to confirm untracking descendants
	// Returns true to proceed with untracking all descendants, false to cancel
	PromptConfirmUntrackDescendants(branchName string, descendantCount int) (bool, error)

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool
}

// NullHandler is a no-op handler for when nil is passed
type NullHandler struct{}

// PromptConfirmUntrackDescendants implements Handler. Returns true (auto-confirm) for null handler.
func (h *NullHandler) PromptConfirmUntrackDescendants(_ string, _ int) (bool, error) {
	return true, nil
}

// Cleanup implements Handler.
func (h *NullHandler) Cleanup() {}

// IsInteractive implements Handler.
func (h *NullHandler) IsInteractive() bool { return false }
