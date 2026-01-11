// Package tui provides terminal UI utilities.
package tui

// ProgressMsg updates progress state.
// Use this to communicate progress updates to TUI models.
type ProgressMsg struct {
	Completed int // Number of completed operations
	Total     int // Total number of operations
}

// Percent returns the completion percentage (0.0 to 1.0).
// Returns 0 if Total is 0.
func (m ProgressMsg) Percent() float64 {
	if m.Total == 0 {
		return 0
	}
	return float64(m.Completed) / float64(m.Total)
}

// PhaseMsg indicates a phase state change.
// Use this to communicate phase transitions to TUI models.
type PhaseMsg struct {
	Phase  string // Name of the phase
	Status Status // Current status of the phase
	Detail string // Optional detail message
}

// CompleteMsg signals operation completion.
// Use this to indicate that the overall operation is done.
type CompleteMsg struct {
	Success bool   // True if operation succeeded
	Summary string // Summary message to display
	Error   error  // Error if Success is false
}
