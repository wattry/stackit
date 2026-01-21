package dashboard

import (
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

// Color constants for dashboard-specific colors not in the style package.
const (
	// colorBackground is a dark gray for selected row backgrounds
	colorBackground = "236"
	// colorWhite is white for text on dark backgrounds
	colorWhite = "15"
	// colorBlack is black for text on light backgrounds
	colorBlack = "0"
	// colorGray is medium gray for disabled text
	colorGray = "7"
	// colorDarkGray is dark gray for disabled backgrounds
	colorDarkGray = "8"
)

// Centralized styles for the dashboard TUI.
// These use the style package helpers for consistency with other TUIs.
var (
	commonStyles = style.DefaultCommonStyles()

	// Dashboard-specific styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.ColorAccent)).
			Bold(true).
			Padding(0, 1)

	headerStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(style.ColorDimValue)).
				Padding(0, 1)

	paneHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(style.ColorAccent))

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(colorBackground)).
				Bold(true)

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(style.ColorAccent)).
			MarginBottom(1)

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(style.ColorMerged)).
				MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.ColorWarningAlt)).
			Width(12)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray))

	errorTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.ColorErrorAlt))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.ColorDimValue)).
			Padding(0, 1)

	// Badge styles for status indicators
	badgeReady = lipgloss.NewStyle().
			Background(lipgloss.Color(style.ColorSuccessAlt)).
			Foreground(lipgloss.Color(colorBlack)).
			Padding(0, 1)

	badgePending = lipgloss.NewStyle().
			Background(lipgloss.Color(style.ColorWarningAlt)).
			Foreground(lipgloss.Color(colorBlack)).
			Padding(0, 1)

	badgeBlocked = lipgloss.NewStyle().
			Background(lipgloss.Color(style.ColorErrorAlt)).
			Foreground(lipgloss.Color(colorWhite)).
			Padding(0, 1)

	badgeIncomplete = lipgloss.NewStyle().
			Background(lipgloss.Color(colorDarkGray)).
			Foreground(lipgloss.Color(colorWhite)).
			Padding(0, 1)

	// Button styles
	buttonPrimary = lipgloss.NewStyle().
			Background(lipgloss.Color(style.ColorSuccessAlt)).
			Foreground(lipgloss.Color(colorBlack)).
			Padding(0, 2)

	// Pane styles (moved from inline)
	headerBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false)

	leftPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			Padding(0, 1)

	rightPaneStyle = lipgloss.NewStyle().
			Padding(0, 1)

	cartPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			Padding(0, 1)

	countBadgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(style.ColorAccent)).
			Foreground(lipgloss.Color(colorBlack)).
			Padding(0, 1)

	dialogStyle = lipgloss.NewStyle().
			Padding(2, 4)
)
