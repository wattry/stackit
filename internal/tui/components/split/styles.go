package split

import (
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

// Styles contains styling for the split component
type Styles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Selected    lipgloss.Style
	Unselected  lipgloss.Style
	Cursor      lipgloss.Style
	Description lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Hint        lipgloss.Style
	StatusIcons style.StatusIcons
	Common      style.CommonStyles
	Status      style.StatusStyles
}

// DefaultStyles returns the default styles for the split component
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")),
		Unselected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")),
		Cursor: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),
		Description: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")),
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")),
		StatusIcons: style.DefaultStatusIcons(),
		Common:      style.DefaultCommonStyles(),
		Status:      style.DefaultStatusStyles(),
	}
}
