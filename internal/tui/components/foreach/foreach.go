// Package foreach provides a TUI component for displaying the progress of foreach command execution.
package foreach

import (
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

// Item represents a branch being processed
type Item struct {
	BranchName string
	Status     string // "pending", "running", "done", "error"
	Output     string
	Error      error
}

// Styles defines the visual styling for the foreach component.
// It uses the shared style definitions from internal/tui/style for consistency.
type Styles struct {
	SpinnerStyle lipgloss.Style
	DoneStyle    lipgloss.Style
	ErrorStyle   lipgloss.Style
	BranchStyle  lipgloss.Style
	OutputStyle  lipgloss.Style
	DimStyle     lipgloss.Style
}

// DefaultStyles returns the default styles for the foreach component,
// using shared styles from the style package for consistency.
func DefaultStyles() Styles {
	statusStyles := style.DefaultStatusStyles()
	commonStyles := style.DefaultCommonStyles()
	return Styles{
		SpinnerStyle: commonStyles.Spinner,
		DoneStyle:    statusStyles.Done,
		ErrorStyle:   statusStyles.Error,
		BranchStyle:  commonStyles.Branch.Bold(true),
		OutputStyle:  commonStyles.Dim,
		DimStyle:     commonStyles.Subtle,
	}
}

const (
	// StatusRunning indicates the branch is currently executing
	StatusRunning = "running"
	// StatusDone indicates the branch execution was successful
	StatusDone = "done"
	// StatusError indicates the branch execution failed
	StatusError = "error"
	// StatusPending indicates the branch is waiting to be executed
	StatusPending = "pending"
)
