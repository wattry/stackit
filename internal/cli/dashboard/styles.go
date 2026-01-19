package dashboard

import (
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

// Centralized styles for the dashboard TUI.
// These use the style package helpers for consistency with other TUIs.
var (
	commonStyles = style.DefaultCommonStyles()

	// Dashboard-specific styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true).
			Padding(0, 1)

	headerStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(0, 1)

	paneHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(12)

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236"))

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			MarginBottom(1)

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5")).
				MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Width(12)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	errorTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
)
