package dashboard

import (
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

// Color palette for visual hierarchy (following lipgloss best practices)
var (
	// Adaptive colors for light/dark mode support
	accent = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
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

	// Badge styles for status indicators
	badgeReady = lipgloss.NewStyle().
			Background(lipgloss.Color("2")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1)

	badgePending = lipgloss.NewStyle().
			Background(lipgloss.Color("3")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1)

	badgeBlocked = lipgloss.NewStyle().
			Background(lipgloss.Color("1")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	badgeIncomplete = lipgloss.NewStyle().
			Background(lipgloss.Color("8")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	// Button styles
	buttonPrimary = lipgloss.NewStyle().
			Background(accent).
			Foreground(lipgloss.Color("0")).
			Padding(0, 2)

	buttonDisabled = lipgloss.NewStyle().
			Background(lipgloss.Color("8")).
			Foreground(lipgloss.Color("7")).
			Padding(0, 2)
)
