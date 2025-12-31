package sync

// Phase represents the current phase of the sync operation
type Phase string

// Phases of the sync operation
const (
	PhaseTrunk   Phase = "trunk"
	PhaseGitHub  Phase = "github"
	PhaseClean   Phase = "clean"
	PhaseRestack Phase = "restack"
)

// EventType represents the type of sync event
type EventType string

// Event types for sync operations
const (
	EventStarted   EventType = "started"
	EventProgress  EventType = "progress"
	EventCompleted EventType = "completed"
	EventSkipped   EventType = "skipped"
)

// Event represents a progress update during sync
type Event struct {
	Phase       Phase     // Current phase
	Type        EventType // Event type
	Branch      string    // Branch name (if applicable)
	PRNumber    *int      // PR number (if applicable)
	Message     string    // Human-readable description
	OldRevision string    // For position changes
	NewRevision string    // For position changes
	Conflict    bool      // Is this a conflict?
	Error       error     // If non-nil, this step had an error
}

// Summary holds aggregate results from a sync operation
type Summary struct {
	TrunkUpdated      bool     // Was trunk updated?
	TrunkRevision     string   // New trunk revision (short hash)
	BranchesSynced    int      // Number of branches synced from remote
	BranchesRestacked int      // Number of branches restacked
	BranchesDeleted   int      // Number of branches deleted
	BranchesSkipped   int      // Number of branches skipped (due to conflicts)
	ConflictBranches  []string // Names of branches that conflicted
	UpToDate          bool     // Everything was already current
}

// HasChanges returns true if any operations were performed
func (s *Summary) HasChanges() bool {
	return s.TrunkUpdated || s.BranchesSynced > 0 || s.BranchesRestacked > 0 ||
		s.BranchesDeleted > 0 || s.BranchesSkipped > 0
}

// Handler abstracts TTY vs non-TTY output for sync operations
// It embeds RestackHandler to provide a unified interface for operations that include restacking
type Handler interface {
	// Start is called at the beginning of sync with the total operation count
	Start(totalOps int)

	// EmitEvent is called for each progress update
	EmitEvent(event Event)

	// Complete is called when sync finishes with the summary
	Complete(summary Summary)

	// RestackHandler methods are available for restack-specific output
	// This allows the same handler to be used for standalone restack operations
	RestackHandler
}

// RestackHandler abstracts TTY vs non-TTY output for restack operations
// This is a subset of Handler that can be used independently for the restack command
type RestackHandler interface {
	// OnRestackStart is called at the beginning of restack with branch count
	OnRestackStart(branchCount int)

	// OnRestackBranch is called for each branch during restack
	OnRestackBranch(branch string, result RestackResult, newRev string, prNumber *int)

	// OnRestackComplete is called when restack finishes
	OnRestackComplete(restacked, skipped int, conflicts []string)
}

// RestackResult represents the outcome of a restack operation for a single branch
type RestackResult string

const (
	// RestackDone indicates the branch was successfully restacked
	RestackDone RestackResult = "done"
	// RestackUnneeded indicates the branch didn't need restacking
	RestackUnneeded RestackResult = "unneeded"
	// RestackConflict indicates the branch had a conflict
	RestackConflict RestackResult = "conflict"
)

// NullHandler is a no-op handler for testing or when output is not needed
type NullHandler struct{}

// Start implements Handler.
func (h *NullHandler) Start(_ int) {}

// EmitEvent implements Handler.
func (h *NullHandler) EmitEvent(_ Event) {}

// Complete implements Handler.
func (h *NullHandler) Complete(_ Summary) {}

// OnRestackStart implements RestackHandler.
func (h *NullHandler) OnRestackStart(_ int) {}

// OnRestackBranch implements RestackHandler.
func (h *NullHandler) OnRestackBranch(_ string, _ RestackResult, _ string, _ *int) {}

// OnRestackComplete implements RestackHandler.
func (h *NullHandler) OnRestackComplete(_, _ int, _ []string) {}
