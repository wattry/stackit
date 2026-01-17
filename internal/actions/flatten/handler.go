package flatten

// Step represents a step in the flatten process
type Step string

// Flatten step constants
const (
	StepAnalyzing  Step = "analyzing"
	StepValidating Step = "validating"
	StepFlattening Step = "flattening"
	StepRestacking Step = "restacking"
)

// StepStatus represents the status of a step
type StepStatus string

// Step status constants
const (
	StatusStarted   StepStatus = "started"
	StatusCompleted StepStatus = "completed"
	StatusSkipped   StepStatus = "skipped"
	StatusFailed    StepStatus = "failed"
)

// PlannedMove represents a single branch move in the flatten plan
type PlannedMove struct {
	Branch    string // Branch being moved
	OldParent string // Current parent branch
	NewParent string // Target parent branch (closer to trunk)
}

// Preview contains information about the planned flatten for confirmation
type Preview struct {
	Moves          []PlannedMove // Branches that will be moved
	UnchangedCount int           // Number of branches that won't change
	HasConflicts   bool          // Whether the flatten will cause conflicts
	ConflictBranch string        // Which branch would have conflicts (if any)
	ConflictError  string        // Error message describing the conflict
}

// Result contains the result of the flatten action
type Result struct {
	MovedCount     int // Number of branches that were moved
	UnchangedCount int // Number of branches that stayed in place
}

// Handler receives events from flatten action
type Handler interface {
	// Start is called at the beginning of flatten
	Start(branchCount int)

	// OnStep is called for each step in the flatten process
	OnStep(step Step, status StepStatus, message string)

	// OnBranchMoved is called when a branch is moved to a new parent
	OnBranchMoved(branch, oldParent, newParent string)

	// Complete is called when flatten finishes
	Complete(result Result)

	// Cleanup restores terminal state (may be no-op)
	Cleanup()

	// IsInteractive returns true if the handler supports interactive prompts
	IsInteractive() bool

	// PromptConfirmFlatten displays a preview of the flatten and asks for confirmation.
	// Returns true to proceed with the flatten, false to cancel.
	// In non-interactive mode, returns true (auto-confirm).
	PromptConfirmFlatten(preview Preview) (bool, error)
}

// NullHandler is a no-op handler for when nil is passed
type NullHandler struct{}

// Start implements Handler.
func (h *NullHandler) Start(_ int) {}

// OnStep implements Handler.
func (h *NullHandler) OnStep(_ Step, _ StepStatus, _ string) {}

// OnBranchMoved implements Handler.
func (h *NullHandler) OnBranchMoved(_, _, _ string) {}

// Complete implements Handler.
func (h *NullHandler) Complete(_ Result) {}

// Cleanup implements Handler.
func (h *NullHandler) Cleanup() {}

// IsInteractive implements Handler.
func (h *NullHandler) IsInteractive() bool { return false }

// PromptConfirmFlatten implements Handler. Returns true (auto-confirm) for null handler.
func (h *NullHandler) PromptConfirmFlatten(_ Preview) (bool, error) { return true, nil }
