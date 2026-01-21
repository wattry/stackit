// Package scope implements the stackit scope command for managing branch scopes.
package scope

// Handler receives events from scope action
type Handler interface {
	// PromptConfirmRename prompts user to confirm branch rename after scope change
	// Returns true to proceed with rename, false to skip
	PromptConfirmRename(branchName, oldScope, newScope string) (bool, error)

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool
}

// NullHandler is a no-op handler for when nil is passed
type NullHandler struct{}

// PromptConfirmRename implements Handler. Returns false (skip rename) for null handler.
func (h *NullHandler) PromptConfirmRename(_, _, _ string) (bool, error) { return false, nil }

// Cleanup implements Handler.
func (h *NullHandler) Cleanup() {}

// IsInteractive implements Handler.
func (h *NullHandler) IsInteractive() bool { return false }
