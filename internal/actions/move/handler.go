package move

import "stackit.dev/stackit/internal/actions/handler"

// Step represents a step in the move process
type Step string

// Move step constants
const (
	StepValidating  Step = "validating"
	StepReparenting Step = "reparenting"
	StepRestacking  Step = "restacking"
)

// Result contains the result of the move action
type Result struct {
	SourceBranch string
	OldParent    string
	NewParent    string
	Renamed      bool
	NewName      string
}

// Preview contains information about the planned move for confirmation
type Preview struct {
	SourceBranch     string   // Branch being moved
	OldParent        string   // Current parent branch
	NewParent        string   // Target parent branch
	Commits          []string // Commit subjects that will be moved
	Descendants      []string // Descendant branches that will be restacked
	HasConflicts     bool     // Whether the move will cause conflicts
	ConflictBranch   string   // Which branch would have conflicts (if any)
	ConflictError    string   // Error message describing the conflict
	ConflictingFiles []string // Files that have conflicts (if any)
}

// Handler receives events from move action
type Handler interface {
	// Start is called at the beginning of move
	Start(sourceBranch, oldParent, newParent string)

	// OnStep is called for each step in the move process
	OnStep(step Step, status handler.StepStatus, message string)

	// OnRename is called when a branch is renamed due to scope change
	OnRename(oldName, newName string)

	// Complete is called when move finishes
	Complete(result Result)

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool

	// PromptRename prompts user to confirm branch rename due to scope change
	PromptRename(oldName, oldScope, newScope string) (bool, error)

	// PromptConfirmMove displays a preview of the move and asks for confirmation.
	// Returns true to proceed with the move, false to cancel.
	// In non-interactive mode, returns true (auto-confirm).
	PromptConfirmMove(preview Preview) (bool, error)
}

// NullHandler is a no-op handler for when nil is passed.
// It embeds handler.NullBase for Cleanup() and IsInteractive().
type NullHandler struct {
	handler.NullBase
	handler.NullProgress[Step]
}

// Start implements Handler.
func (h *NullHandler) Start(string, string, string) {}

// OnRename implements Handler.
func (h *NullHandler) OnRename(string, string) {}

// Complete implements Handler.
func (h *NullHandler) Complete(Result) {}

// PromptRename implements Handler.
func (h *NullHandler) PromptRename(string, string, string) (bool, error) { return false, nil }

// PromptConfirmMove implements Handler. Returns true (auto-confirm) for null handler.
func (h *NullHandler) PromptConfirmMove(Preview) (bool, error) { return true, nil }
