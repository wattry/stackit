// Package get provides types and interfaces for the get command.
package get

import (
	"stackit.dev/stackit/internal/handlers"
)

// Phase represents the current phase of the get operation
type Phase string

// Phases of the get operation
const (
	PhaseFetch    Phase = "fetch"    // Fetching branches from remote
	PhaseSync     Phase = "sync"     // Syncing branches (create/update)
	PhaseMetadata Phase = "metadata" // Fetching and applying metadata
	PhaseCheckout Phase = "checkout" // Checking out target branch
)

// EventType represents the type of get event
type EventType string

// Event types for get operations
const (
	EventStarted   EventType = "started"
	EventProgress  EventType = "progress"
	EventCompleted EventType = "completed"
	EventSkipped   EventType = "skipped"
)

// Event represents a progress update during get
type Event struct {
	Phase       Phase     // Current phase
	Type        EventType // Event type
	Branch      string    // Branch name (if applicable)
	PRNumber    *int      // PR number (if applicable)
	Message     string    // Human-readable description
	NewRevision string    // For position changes
	IsNew       bool      // Is this a new branch?
	Error       error     // If non-nil, this step had an error
}

// Summary holds aggregate results from a get operation
type Summary struct {
	TargetBranch    string // The branch that was retrieved
	BranchesCreated int    // Number of branches created
	BranchesUpdated int    // Number of branches updated
	Restacked       int    // Number of branches restacked
	IsFrozen        bool   // Was the target branch frozen?
	UpToDate        bool   // Everything was already current
}

// Handler abstracts TTY vs non-TTY output for get operations
// It embeds RestackHandler to provide consistent output for restack phase
type Handler interface {
	// Start is called at the beginning of get with target info
	Start(targetBranch string, prNumber *int)

	// EmitEvent is called for each progress update
	EmitEvent(event Event)

	// Complete is called when get finishes with the summary
	Complete(summary Summary)

	// RestackHandler methods are available for restack phase output
	// This ensures consistent restack output between get, sync, and restack commands
	handlers.RestackHandler
}

// NullHandler is a no-op handler for testing or when output is not needed
type NullHandler struct{}

// Start implements Handler.
func (h *NullHandler) Start(_ string, _ *int) {}

// EmitEvent implements Handler.
func (h *NullHandler) EmitEvent(_ Event) {}

// Complete implements Handler.
func (h *NullHandler) Complete(_ Summary) {}

// OnRestackStart implements RestackHandler.
func (h *NullHandler) OnRestackStart(_ int) {}

// OnRestackBranch implements RestackHandler.
func (h *NullHandler) OnRestackBranch(_ string, _ handlers.RestackResult, _ string, _ *int) {}

// OnRestackComplete implements RestackHandler.
func (h *NullHandler) OnRestackComplete(_, _ int, _ []string) {}
