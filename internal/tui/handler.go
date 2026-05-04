// Package tui provides terminal UI utilities.
package tui

import tea "charm.land/bubbletea/v2"

// Sender is the interface for sending messages to a TUI.
// Both *Runner and *MockRunner implement this interface.
type Sender interface {
	// Send sends a message to the TUI model.
	Send(msg tea.Msg)

	// Pause pauses TUI rendering for interactive prompts.
	Pause()

	// Resume resumes TUI rendering after a pause.
	Resume()

	// Wait blocks until the TUI exits.
	Wait()

	// Cleanup performs terminal cleanup.
	Cleanup()

	// IsHealthy returns true if the TUI is running and responsive.
	IsHealthy() bool
}

// Handler is the base interface for all TUI handlers.
// Handlers abstract the difference between interactive (TTY) and non-interactive output.
type Handler interface {
	// IsInteractive returns true if this handler supports interactive prompts.
	// Interactive handlers can pause the TUI for user input.
	IsInteractive() bool

	// Cleanup performs any necessary cleanup. Safe to call multiple times.
	// For interactive handlers, this typically delegates to the Runner.
	Cleanup()
}

// ProgressHandler extends Handler with progress tracking.
// Use this for operations that have a known number of steps.
type ProgressHandler interface {
	Handler

	// Start initializes progress tracking with total operations.
	// Call this at the beginning of the operation.
	Start(totalOps int)

	// Progress updates the current progress.
	// completed should be <= total passed to Start().
	Progress(completed, total int)

	// Complete signals completion with a summary message.
	// Call this when the operation is done.
	Complete(summary string)
}

// PhaseHandler extends ProgressHandler with phase tracking.
// Use this for operations that have distinct phases (e.g., sync: trunk, github, clean, restack).
type PhaseHandler interface {
	ProgressHandler

	// StartPhase begins a named phase.
	// Only one phase can be active at a time.
	StartPhase(phase string)

	// PhaseDetail adds a detail line to the current phase.
	// Use this for per-item progress within a phase.
	PhaseDetail(phase, detail string)

	// CompletePhase marks a phase as done.
	// Call this before starting the next phase.
	CompletePhase(phase string)
}

// PromptHandler provides interactive prompt capabilities.
// Handlers that support prompts should implement this interface.
type PromptHandler interface {
	Handler

	// SupportsPrompts returns true if this handler can show prompts.
	// This is typically the same as IsInteractive(), but allows for
	// handlers that are interactive but don't support prompts.
	SupportsPrompts() bool
}
