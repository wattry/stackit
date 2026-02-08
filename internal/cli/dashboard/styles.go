package dashboard

import "stackit.dev/stackit/internal/tui/style"

// All dashboard styles are defined in style.DefaultDashboardStyles() so that
// the st-tui storyboard stays in sync with the real dashboard.
var ds = style.DefaultDashboardStyles()

// Re-export for convenient access within this package.
var (
	commonStyles = style.DefaultCommonStyles()

	titleStyle        = ds.Title
	headerStatusStyle = ds.HeaderStatus
	headerBorderStyle = ds.HeaderBorder
	paneHeaderStyle   = ds.PaneHeader
	selectedRowStyle  = ds.SelectedRow
	footerStyle       = ds.Footer
	errorTextStyle    = ds.ErrorText

	leftPaneStyle  = ds.LeftPane
	rightPaneStyle = ds.RightPane
	actionBarStyle = ds.ActionBar

	badgeReady      = ds.BadgeReady
	badgePending    = ds.BadgePending
	badgeBlocked    = ds.BadgeBlocked
	badgeIncomplete = ds.BadgeIncomplete

	buttonPrimary = ds.ButtonPrimary

	helpTitleStyle   = ds.HelpTitle
	helpSectionStyle = ds.HelpSection
	helpKeyStyle     = ds.HelpKey
	helpDescStyle    = ds.HelpDesc

	dialogStyle = ds.Dialog
)
