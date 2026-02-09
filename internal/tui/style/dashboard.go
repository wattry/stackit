package style

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Column width constants for alignment in the stack list.
// These ensure columns stay aligned regardless of unicode character widths.
const (
	// CheckboxColumnWidth is the display width for the checkbox column.
	// Widest value is "[🔒]" which renders as 4 display columns.
	CheckboxColumnWidth = 4

	// StatusIconColumnWidth is the display width for the status icon column.
	// Widest value is "⏳" which renders as 2 display columns.
	StatusIconColumnWidth = 2
)

// PadToWidth pads a rendered string with spaces to reach targetWidth display columns.
// It uses lipgloss.Width() to measure actual display width, handling unicode correctly.
func PadToWidth(rendered string, targetWidth int) string {
	actual := lipgloss.Width(rendered)
	if actual >= targetWidth {
		return rendered
	}
	return rendered + strings.Repeat(" ", targetWidth-actual)
}

// Dashboard color constants shared between the real dashboard and story previews.
const (
	// ColorDashboardBackground is a dark gray for selected row backgrounds.
	ColorDashboardBackground = "236"
	// ColorDashboardWhite is white for text on dark backgrounds.
	ColorDashboardWhite = "15"
	// ColorDashboardBlack is black for text on light backgrounds.
	ColorDashboardBlack = "0"
	// ColorDashboardGray is medium gray for disabled text.
	ColorDashboardGray = "7"
	// ColorDashboardDarkGray is dark gray for disabled backgrounds.
	ColorDashboardDarkGray = "8"
	// ColorDashboardBorder is dim gray for panel borders.
	ColorDashboardBorder = "240"
	// ColorDashboardFocusedBorder is brighter for the focused pane border.
	ColorDashboardFocusedBorder = "252"
)

// DashboardStyles holds all styles for the shippable work dashboard.
// Both the real dashboard and st-tui stories use these to stay in sync.
type DashboardStyles struct {
	Title        lipgloss.Style
	HeaderStatus lipgloss.Style
	HeaderBorder lipgloss.Style
	PaneHeader   lipgloss.Style
	SelectedRow  lipgloss.Style
	Footer       lipgloss.Style
	ErrorText    lipgloss.Style

	// Pane styles
	LeftPane  lipgloss.Style
	RightPane lipgloss.Style
	ActionBar lipgloss.Style

	// Badge styles
	BadgeReady      lipgloss.Style
	BadgePending    lipgloss.Style
	BadgeBlocked    lipgloss.Style
	BadgeIncomplete lipgloss.Style

	// Button styles
	ButtonPrimary lipgloss.Style

	// Help styles
	HelpTitle   lipgloss.Style
	HelpSection lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style

	// Dialog
	Dialog lipgloss.Style
}

// DefaultDashboardStyles returns the canonical dashboard styles.
func DefaultDashboardStyles() DashboardStyles {
	return DashboardStyles{
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
			Bold(true).
			Padding(0, 1),

		HeaderStatus: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDimValue)).
			Padding(0, 1),

		HeaderBorder: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color(ColorDashboardBorder)),

		PaneHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorAccent)),

		SelectedRow: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorDashboardBackground)).
			Bold(true),

		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDimValue)).
			Padding(0, 1),

		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorErrorAlt)),

		// Pane styles
		LeftPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorDashboardBorder)).
			Padding(1),

		RightPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorDashboardBorder)).
			Padding(1),

		ActionBar: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color(ColorDashboardBorder)).
			Padding(0, 1),

		// Badge styles
		BadgeReady: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorSuccessAlt)).
			Foreground(lipgloss.Color(ColorDashboardBlack)).
			Padding(0, 1),

		BadgePending: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorWarningAlt)).
			Foreground(lipgloss.Color(ColorDashboardBlack)).
			Padding(0, 1),

		BadgeBlocked: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorErrorAlt)).
			Foreground(lipgloss.Color(ColorDashboardWhite)).
			Padding(0, 1),

		BadgeIncomplete: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorDashboardDarkGray)).
			Foreground(lipgloss.Color(ColorDashboardWhite)).
			Padding(0, 1),

		// Button styles
		ButtonPrimary: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorSuccessAlt)).
			Foreground(lipgloss.Color(ColorDashboardBlack)).
			Padding(0, 2),

		// Help styles
		HelpTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorAccent)).
			MarginBottom(1),

		HelpSection: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorMerged)).
			MarginTop(1),

		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarningAlt)).
			Width(12),

		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDashboardGray)),

		// Dialog
		Dialog: lipgloss.NewStyle().
			Padding(2, 4),
	}
}
