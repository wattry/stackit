// Package tui provides terminal UI utilities.
package tui

// Status represents the state of an operation or phase.
// This provides a unified status type across all TUI components.
type Status string

const (
	// StatusPending indicates the operation has not started.
	StatusPending Status = "pending"

	// StatusActive indicates the operation is in progress.
	StatusActive Status = "active"

	// StatusDone indicates the operation completed successfully.
	StatusDone Status = "done"

	// StatusError indicates the operation failed.
	StatusError Status = "error"

	// StatusSkipped indicates the operation was skipped.
	StatusSkipped Status = "skipped"

	// StatusWaiting indicates the operation is waiting for user input or external event.
	StatusWaiting Status = "waiting"
)

// IsTerminal returns true if this status represents a final state
// (done, error, or skipped).
func (s Status) IsTerminal() bool {
	return s == StatusDone || s == StatusError || s == StatusSkipped
}

// IsActive returns true if the status indicates active work.
func (s Status) IsActive() bool {
	return s == StatusActive || s == StatusWaiting
}

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}
