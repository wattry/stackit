package style

import "github.com/charmbracelet/lipgloss"

// StatusStyles contains styles for status indicators used across TUI components.
// This provides a consistent look for pending, active, done, and error states.
type StatusStyles struct {
	// Pending style for items not yet started
	Pending lipgloss.Style
	// Active style for items currently in progress
	Active lipgloss.Style
	// Done style for successfully completed items
	Done lipgloss.Style
	// Error style for failed items
	Error lipgloss.Style
	// Warning style for items needing attention
	Warning lipgloss.Style
}

// StatusIcons contains icon characters for status indicators.
type StatusIcons struct {
	Pending string
	Active  string
	Done    string
	Error   string
	Warning string
}

// DefaultStatusStyles returns the standard status styles for stackit TUIs.
func DefaultStatusStyles() StatusStyles {
	return StatusStyles{
		Pending: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Active:  lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		Done:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	}
}

// DefaultStatusIcons returns the standard status icons for stackit TUIs.
func DefaultStatusIcons() StatusIcons {
	return StatusIcons{
		Pending: "○",
		Active:  "◐",
		Done:    "✓",
		Error:   "✗",
		Warning: "⚠",
	}
}

// CommonStyles contains commonly used text styles.
type CommonStyles struct {
	// Bold style for emphasized text
	Bold lipgloss.Style
	// Dim style for de-emphasized text (adaptive to terminal background)
	Dim lipgloss.Style
	// Subtle style for secondary text (adaptive to terminal background)
	Subtle lipgloss.Style
	// Branch style for branch names
	Branch lipgloss.Style
	// URL style for links
	URL lipgloss.Style
	// Spinner style for animated spinners
	Spinner lipgloss.Style
}

// DefaultCommonStyles returns the standard common styles for stackit TUIs.
func DefaultCommonStyles() CommonStyles {
	return CommonStyles{
		Bold:    lipgloss.NewStyle().Bold(true),
		Dim:     DimStyle(),
		Subtle:  SubtleStyle(),
		Branch:  BranchStyle(false, false, false),
		URL:     DimStyle(),
		Spinner: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
	}
}
