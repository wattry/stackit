// Package navigation implements the stackit top/bottom commands for navigating stacked branches.
package navigation

// Handler receives events from navigation action
type Handler interface {
	// PromptSelectBranch prompts user to select a branch when multiple children exist
	// Returns the selected branch name
	PromptSelectBranch(message string, branches []string) (string, error)

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool
}

// NullHandler is a no-op handler for when nil is passed
type NullHandler struct{}

// PromptSelectBranch implements Handler. Returns empty string for null handler.
func (h *NullHandler) PromptSelectBranch(_ string, _ []string) (string, error) { return "", nil }

// Cleanup implements Handler.
func (h *NullHandler) Cleanup() {}

// IsInteractive implements Handler.
func (h *NullHandler) IsInteractive() bool { return false }
