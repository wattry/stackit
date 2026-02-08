package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/shippable"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
)

// View renders the dashboard.
func (m *shippableModel) View() string {
	switch m.state {
	case stateHelp:
		return m.renderHelp()
	case stateConfirming:
		return m.renderConfirmation()
	default:
		// All other states (including loading) show the main view
		return m.renderMain()
	}
}

// renderMain renders the main dashboard view.
func (m *shippableModel) renderMain() string {
	// Build sections first, then calculate heights dynamically
	header := m.renderHeader()
	footer := m.renderFooter()

	// Calculate heights using lipgloss.Height() for accurate measurement
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)

	// Action bar height (only if items selected)
	// Fixed height: border(1) + content(1) + padding(1)
	actionBarHeight := 0
	if m.cache.selectedCount > 0 {
		actionBarHeight = 3
	}

	contentHeight := m.Height - headerHeight - footerHeight - actionBarHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	leftWidth := m.Width / 2
	rightWidth := m.Width - leftWidth

	// Build panes with calculated dimensions
	leftPane := m.renderStackList(leftWidth, contentHeight)
	rightPane := m.renderDetailsPanel(rightWidth, contentHeight)

	// Combine stack viewer panes (top row)
	stackViewer := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Build final layout
	var sections []string
	sections = append(sections, header, stackViewer)

	// Add action bar if items selected (bottom row)
	if m.cache.selectedCount > 0 {
		bar := m.renderActionBar(m.Width)
		sections = append(sections, bar)
	}

	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the dashboard header.
func (m *shippableModel) renderHeader() string {
	title := titleStyle.Render("SHIPPABLE WORK")

	var status string
	switch {
	case m.state == stateLoading:
		status = headerStatusStyle.Render(m.Spinner.View() + " Loading...")
	case m.state == stateAnalyzing:
		status = headerStatusStyle.Render(m.Spinner.View() + " Analyzing...")
	case m.state == stateShipping:
		status = headerStatusStyle.Render(m.Spinner.View() + " Shipping...")
	case m.state == statePublishing:
		status = headerStatusStyle.Render(m.Spinner.View() + " Publishing...")
	case m.errorMessage != "":
		status = errorTextStyle.Render(m.errorMessage)
	case m.statusMessage != "":
		status = headerStatusStyle.Render(m.statusMessage)
	case m.analysis != nil:
		status = headerStatusStyle.Render(fmt.Sprintf("%d stacks (%d shippable)",
			m.analysis.TotalStacks(), m.analysis.ShippableCount))
	}

	left := lipgloss.JoinHorizontal(lipgloss.Left, title, "  ", status)

	// Show refresh countdown on the right
	var refreshStatus string
	if !m.lastRefresh.IsZero() && m.state == stateMain {
		timeSinceRefresh := time.Since(m.lastRefresh)
		timeUntilRefresh := autoRefreshInterval - timeSinceRefresh
		if timeUntilRefresh < 0 {
			timeUntilRefresh = 0
		}
		secondsUntil := int(timeUntilRefresh.Seconds())
		refreshStatus = style.ColorDim(fmt.Sprintf("refresh in %ds", secondsUntil))
	}

	if refreshStatus != "" {
		gap := m.Width - lipgloss.Width(left) - lipgloss.Width(refreshStatus) - 2
		if gap < 2 {
			gap = 2
		}
		left = left + strings.Repeat(" ", gap) + refreshStatus
	}

	return headerBorderStyle.
		Width(m.Width).
		Render(left)
}

// renderStackList renders the list of stacks.
func (m *shippableModel) renderStackList(width, height int) string {
	// Width/Height include padding but not borders, so subtract border size only
	borderW := leftPaneStyle.GetHorizontalBorderSize()
	borderH := leftPaneStyle.GetVerticalBorderSize()
	paneStyle := leftPaneStyle.
		Width(width - borderW).
		Height(height - borderH)

	var sb strings.Builder
	sb.WriteString(paneHeaderStyle.Render("STACKS") + "\n\n")

	// Show appropriate content based on state
	if len(m.stacks) == 0 {
		if m.state == stateLoading {
			sb.WriteString(commonStyles.Dim.Render("Loading stacks..."))
		} else {
			sb.WriteString(commonStyles.Dim.Render("No stacks found. Create one with `stackit create`"))
		}
		// Add help hints at bottom even when empty
		sb.WriteString("\n\n" + style.ColorDim("↑/↓ navigate"))
		return paneStyle.Render(sb.String())
	}

	for i, stack := range m.stacks {
		line := m.renderStackLine(stack, i == m.selectedIndex)
		sb.WriteString(line + "\n")

		// If expanded, show branches
		if m.expanded[stack.RootBranch()] {
			for _, branch := range stack.Stack.AllBranches {
				branchLine := "    " + m.renderBranchLine(branch)
				sb.WriteString(branchLine + "\n")
			}
		}
	}

	// Add contextual shortcuts at bottom of pane
	sb.WriteString("\n" + style.ColorDim("↑/↓ navigate  space select  enter expand  A all"))

	return paneStyle.Render(sb.String())
}

// renderStackLine renders a single stack in the list.
func (m *shippableModel) renderStackLine(stack shippable.Stack, focused bool) string {
	// Cursor indicator for focused row
	cursor := "  "
	if focused {
		cursor = style.ColorCyan("▸ ")
	}

	root := stack.RootBranch()

	// Selection checkbox - show lock icon if locked
	// Pad to fixed width so columns align regardless of emoji width
	var checkbox string
	switch {
	case m.isLocked(root):
		checkbox = style.ColorYellow("[🔒]")
	case m.selected[root]:
		checkbox = style.ColorCyan("[x]")
	default:
		checkbox = "[ ]"
	}
	checkbox = style.PadToWidth(checkbox, style.CheckboxColumnWidth)

	// Status icon - pad to fixed width for alignment
	statusIcon := style.PadToWidth(m.getStatusIcon(stack.Status), style.StatusIconColumnWidth)

	// Stack title: use cached title (computed at refresh time)
	name := m.cache.stackTitles[root]
	if name == "" {
		name = root // Fallback if cache not populated
	}

	// Mark the stack containing the checked-out branch
	isCurrentStack := m.cache.currentStackRoot == root

	// Only show branch count if more than 1 branch
	branchCount := ""
	if count := stack.BranchCount(); count > 1 {
		branchCount = fmt.Sprintf("(%d branches)", count)
	}

	// Expand indicator (only show if stack has multiple branches)
	expandIndicator := ""
	if stack.BranchCount() > 1 {
		expandIndicator = "▶"
		if m.expanded[root] {
			expandIndicator = "▼"
		}
	}

	// Calculate max name length based on available pane width
	// Use lipgloss.Width() for accurate unicode measurement
	paneWidth := m.Width / 2
	overhead := 2 + style.CheckboxColumnWidth + 1 + style.StatusIconColumnWidth + 1 + lipgloss.Width(branchCount) + lipgloss.Width(expandIndicator) + 6
	maxNameLen := paneWidth - overhead
	if maxNameLen < 20 {
		maxNameLen = 20 // Minimum readable length
	}

	// Truncate if needed
	if len(name) > maxNameLen {
		name = name[:maxNameLen-3] + "..."
	}

	// Apply current stack styling to the name
	if isCurrentStack {
		name = style.ColorGreen(name)
	}

	// Build line
	var line string
	if branchCount != "" {
		line = fmt.Sprintf("%s%s %s %s %s %s", cursor, checkbox, statusIcon, name, style.ColorDim(branchCount), expandIndicator)
	} else {
		line = fmt.Sprintf("%s%s %s %s", cursor, checkbox, statusIcon, name)
	}

	// Highlight if focused
	if focused {
		line = selectedRowStyle.Render(line)
	}

	return line
}

// renderBranchLine renders a single branch within an expanded stack.
func (m *shippableModel) renderBranchLine(branchName string) string {
	isCurrent := branchName == m.cache.currentBranch

	// Use cached annotation for PR info
	ann, hasCached := m.cache.branchAnnotations[branchName]

	var prInfo string
	if hasCached && ann.PRNumber != nil && *ann.PRNumber > 0 {
		prInfo = fmt.Sprintf(" #%d", *ann.PRNumber)
	}

	displayName := branchName
	if isCurrent {
		displayName = style.ColorGreen(branchName)
	}

	// Build suffix parts
	var suffixParts []string
	if prInfo != "" {
		suffixParts = append(suffixParts, style.ColorDim(prInfo))
	}
	if blockingStatus := m.formatBranchBlockingStatus(branchName); blockingStatus != "" {
		suffixParts = append(suffixParts, blockingStatus)
	}
	if isCurrent {
		suffixParts = append(suffixParts, style.ColorDim("(current)"))
	}

	suffix := ""
	if len(suffixParts) > 0 {
		suffix = " " + strings.Join(suffixParts, " ")
	}

	return style.ColorDim("├── ") + displayName + suffix
}

// formatBranchBlockingStatus returns a short colored status for a blocked branch.
func (m *shippableModel) formatBranchBlockingStatus(branchName string) string {
	reason, blocked := m.cache.branchBlocking[branchName]
	if !blocked {
		return ""
	}

	switch reason {
	case shippable.ReasonCIFailing:
		return style.ColorRed("✗ CI")
	case shippable.ReasonCIPending:
		return style.ColorYellow("⏳ CI")
	case shippable.ReasonChangesRequested:
		return style.ColorRed("✗ changes requested")
	case shippable.ReasonReviewRequired:
		return style.ColorYellow("○ review needed")
	case shippable.ReasonDraft:
		return style.ColorDim("draft")
	case shippable.ReasonNoPR:
		return style.ColorYellow("no PR")
	case shippable.ReasonNotPushed:
		return style.ColorYellow("not pushed")
	default:
		return ""
	}
}

// getStatusIcon returns the icon for a stack status.
func (m *shippableModel) getStatusIcon(status shippable.Status) string {
	switch status {
	case shippable.StatusShippable:
		return style.ColorGreen("✓")
	case shippable.StatusPending:
		return style.ColorYellow("⏳")
	case shippable.StatusBlocked:
		return style.ColorRed("✗")
	case shippable.StatusIncomplete:
		return style.ColorDim("○")
	default:
		return "?"
	}
}

// renderDetailsPanel renders the right-side details panel (always shows stack details).
func (m *shippableModel) renderDetailsPanel(width, height int) string {
	borderW := rightPaneStyle.GetHorizontalBorderSize()
	borderH := rightPaneStyle.GetVerticalBorderSize()
	paneStyle := rightPaneStyle.
		Width(width - borderW).
		Height(height - borderH)

	var sb strings.Builder
	sb.WriteString(paneHeaderStyle.Render("DETAILS") + "\n\n")

	if m.focusedStack != nil {
		sb.WriteString(m.renderStackDetails(m.focusedStack))
	} else {
		sb.WriteString(commonStyles.Dim.Render("Select a stack to see details"))
	}

	return paneStyle.Render(sb.String())
}

// renderActionBar renders the compact action bar when stacks are selected.
func (m *shippableModel) renderActionBar(width int) string {
	barStyle := actionBarStyle.Width(width - actionBarStyle.GetHorizontalBorderSize())

	// Use cached selected stacks
	selected := m.cache.selectedStacks
	totalBranches := 0
	for _, s := range selected {
		totalBranches += s.BranchCount()
	}

	// Summary text
	summary := fmt.Sprintf("%d stacks selected (%d branches)", len(selected), totalBranches)

	// Actions
	shipAction := buttonPrimary.Render("[s] Ship")

	var analysisAction string
	if m.combination != nil {
		if m.combination.Combinable {
			analysisAction = style.ColorGreen("✓ Compatible")
		} else {
			analysisAction = style.ColorRed("✗ Conflicts")
		}
	} else if len(selected) > 1 {
		analysisAction = style.ColorDim("[a] Analyze")
	}

	line := fmt.Sprintf("%s  %s  %s", summary, shipAction, analysisAction)
	return barStyle.Render(line)
}

// renderStackDetails renders detailed info about a stack with tree view.
func (m *shippableModel) renderStackDetails(stack *shippable.Stack) string {
	var sb strings.Builder

	// Show stack description if present (matches st info --stack)
	if desc := m.cache.stackDescriptions[stack.RootBranch()]; desc != nil {
		// Render title and description together through glamour for consistent formatting
		var markdown string
		if desc.Description != "" {
			markdown = "# " + desc.Title + "\n\n" + desc.Description
		} else {
			markdown = "# " + desc.Title
		}
		rendered := style.RenderMarkdown(markdown)
		sb.WriteString(rendered + "\n")
	}

	// Header: show commit title with status badge, branch name dimmed below
	statusBadge := m.renderStatusBadge(stack.Status)
	// Use cached title (computed at refresh time)
	title := m.cache.stackTitles[stack.RootBranch()]
	if title == "" {
		title = stack.RootBranch() // Fallback if cache not populated
	}
	sb.WriteString(commonStyles.Bold.Render(title) + " " + statusBadge + "\n")
	sb.WriteString(style.ColorDim(stack.RootBranch()) + "\n")

	// Quick stats row
	statsRow := m.renderQuickStats(stack)
	sb.WriteString(statsRow + "\n\n")

	// Stack tree visualization (blocking status is shown inline per branch)
	sb.WriteString(style.ColorDim("Stack:") + "\n")
	treeLines := m.renderStackTree(stack)
	for _, line := range treeLines {
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// renderStatusBadge returns a colored badge for the stack status.
func (m *shippableModel) renderStatusBadge(status shippable.Status) string {
	switch status {
	case shippable.StatusShippable:
		return badgeReady.Render("READY")
	case shippable.StatusPending:
		return badgePending.Render("PENDING")
	case shippable.StatusBlocked:
		return badgeBlocked.Render("BLOCKED")
	case shippable.StatusIncomplete:
		return badgeIncomplete.Render("INCOMPLETE")
	default:
		return ""
	}
}

// renderQuickStats shows a compact row of key stats.
func (m *shippableModel) renderQuickStats(stack *shippable.Stack) string {
	var parts []string

	// Branch count
	parts = append(parts, fmt.Sprintf("%d branches", stack.BranchCount()))

	// Approval status
	if stack.ApprovalOK {
		parts = append(parts, style.ColorGreen("✓ Approved"))
	} else {
		parts = append(parts, style.ColorYellow("○ Review needed"))
	}

	// CI status
	if stack.GitHubCIOK {
		parts = append(parts, style.ColorGreen("✓ CI"))
	} else {
		parts = append(parts, style.ColorRed("✗ CI"))
	}

	return style.ColorDim(strings.Join(parts, " • "))
}

// renderStackTree renders the stack as a tree visualization.
func (m *shippableModel) renderStackTree(stack *shippable.Stack) []string {
	// Use cached tree renderer if available, otherwise create one
	var renderer *tree.StackTreeRenderer
	if m.cache.treeRenderer != nil {
		renderer = m.cache.treeRenderer
	} else {
		// Fallback: create a filtered renderer (shouldn't happen after refresh)
		stackBranches := make(map[string]bool)
		for _, branch := range stack.Stack.AllBranches {
			stackBranches[branch] = true
		}
		renderer = tui.NewStackTreeRendererWithFilter(m.engine, func(branchName string) bool {
			return stackBranches[branchName]
		})
	}

	// Build a map of blocking PRs by branch for quick lookup
	blockingByBranch := make(map[string]*shippable.BlockingPR)
	for i := range stack.BlockingPRs {
		bp := &stack.BlockingPRs[i]
		blockingByBranch[bp.Branch] = bp
	}

	// Add annotations for each branch using cached data
	for _, branchName := range stack.Stack.AllBranches {
		// Start with cached annotation (computed at refresh time)
		ann, hasCached := m.cache.branchAnnotations[branchName]
		if !hasCached {
			continue
		}

		// Overlay blocking status from shippability analysis
		if bp, blocked := blockingByBranch[branchName]; blocked {
			switch bp.Reason {
			case shippable.ReasonCIFailing:
				ann.CheckStatus = tree.CheckStatusFailing
			case shippable.ReasonCIPending:
				ann.CheckStatus = tree.CheckStatusPending
			case shippable.ReasonChangesRequested:
				ann.ReviewStatus = "Changes Requested"
			case shippable.ReasonReviewRequired:
				ann.ReviewStatus = "In Review"
			case shippable.ReasonDraft:
				ann.IsDraft = true
			case shippable.ReasonNoPR:
				ann.CustomLabel = style.ColorYellow("no PR")
			case shippable.ReasonNotPushed:
				ann.CustomLabel = style.ColorYellow("not pushed")
			}
		} else {
			// If not blocked, mark as passing/approved based on stack status
			if stack.GitHubCIOK && ann.CheckStatus == "" {
				ann.CheckStatus = tree.CheckStatusPassing
			}
			if stack.ApprovalOK && ann.ReviewStatus == "" {
				ann.ReviewStatus = "Approved"
			}
		}

		renderer.SetAnnotation(branchName, ann)
	}

	// Render tree in full mode with commit messages to match st info --stack
	opts := tree.RenderOptions{
		Mode:                tree.RenderModeFull,
		HideSummary:         true,
		SkipSelectionPrefix: true,
		ShowCommitMessages:  true,
	}

	return renderer.RenderStack(stack.RootBranch(), opts)
}

// renderFooter renders the footer with global shortcuts.
func (m *shippableModel) renderFooter() string {
	// During async operations, show progress bar
	if m.state == stateLoading || m.state == stateAnalyzing || m.state == stateShipping || m.state == statePublishing {
		return m.renderProgressFooter()
	}

	// Global shortcuts only (pane-specific shortcuts shown in their panes)
	shortcuts := []string{
		"[p] Publish all",
		"[r] Refresh",
		"[?] Help",
		"[q] Quit",
	}
	shortcutsStr := strings.Join(shortcuts, "  ")

	return footerStyle.Width(m.Width).Render(shortcutsStr)
}

// renderProgressFooter renders the footer with a progress bar during async operations.
func (m *shippableModel) renderProgressFooter() string {
	var message string
	switch m.state {
	case stateLoading:
		message = "Refreshing..."
	case stateAnalyzing:
		message = "Analyzing..."
	case stateShipping:
		message = "Shipping..."
	case statePublishing:
		message = "Publishing..."
	}

	if m.progressMessage != "" {
		message = m.progressMessage
	}

	// Build progress line
	var progressLine string
	if m.progressTotal > 0 {
		progressLine = fmt.Sprintf("%s %s (%d/%d)", message, m.progress.View(), m.progressStep, m.progressTotal)
	} else {
		// Show spinner for indeterminate progress
		progressLine = fmt.Sprintf("%s %s", m.Spinner.View(), message)
	}

	quitHint := style.ColorDim("[q] Quit")
	gap := m.Width - lipgloss.Width(progressLine) - lipgloss.Width(quitHint) - 4
	if gap < 2 {
		gap = 2
	}
	line := strings.Repeat(" ", gap)

	return footerStyle.Width(m.Width).Render(progressLine + line + quitHint)
}

// renderHelp renders the help overlay.
func (m *shippableModel) renderHelp() string {
	var sb strings.Builder
	sb.WriteString(helpTitleStyle.Render("Shippable Work Dashboard Help") + "\n\n")

	sb.WriteString(helpSectionStyle.Render("Navigation") + "\n")
	sb.WriteString(helpKeyStyle.Render("j/k, ↑/↓") + helpDescStyle.Render("Move selection up/down") + "\n")
	sb.WriteString(helpKeyStyle.Render("enter") + helpDescStyle.Render("Expand/collapse stack") + "\n")

	sb.WriteString(helpSectionStyle.Render("Selection") + "\n")
	sb.WriteString(helpKeyStyle.Render("space") + helpDescStyle.Render("Toggle stack selection") + "\n")
	sb.WriteString(helpKeyStyle.Render("A") + helpDescStyle.Render("Select all shippable") + "\n")

	sb.WriteString(helpSectionStyle.Render("Actions") + "\n")
	sb.WriteString(helpKeyStyle.Render("s") + helpDescStyle.Render("Ship selected stacks") + "\n")
	sb.WriteString(helpKeyStyle.Render("p") + helpDescStyle.Render("Restack & submit all branches") + "\n")
	sb.WriteString(helpKeyStyle.Render("a") + helpDescStyle.Render("Analyze combination") + "\n")
	sb.WriteString(helpKeyStyle.Render("r") + helpDescStyle.Render("Refresh analysis") + "\n")

	sb.WriteString(helpSectionStyle.Render("Other") + "\n")
	sb.WriteString(helpKeyStyle.Render("?") + helpDescStyle.Render("Toggle this help") + "\n")
	sb.WriteString(helpKeyStyle.Render("q") + helpDescStyle.Render("Quit") + "\n")

	sb.WriteString("\n" + commonStyles.Dim.Render("Press any key to close"))

	return dialogStyle.Render(sb.String())
}

// renderConfirmation renders the ship confirmation dialog.
func (m *shippableModel) renderConfirmation() string {
	// Use cached selected stacks
	selected := m.cache.selectedStacks
	totalBranches := 0
	for _, s := range selected {
		totalBranches += s.BranchCount()
	}

	var sb strings.Builder
	sb.WriteString(helpTitleStyle.Render("SHIP CONFIRMATION") + "\n\n")

	sb.WriteString(fmt.Sprintf("About to ship %d stacks (%d branches total):\n\n",
		len(selected), totalBranches))

	for _, s := range selected {
		sb.WriteString(fmt.Sprintf("  %s (%d branches)\n", s.RootBranch(), s.BranchCount()))
	}

	sb.WriteString("\nThis will:\n")
	sb.WriteString("  - Create consolidation branch\n")
	sb.WriteString("  - Create/update PR to main\n")
	sb.WriteString("  - Merge when CI passes\n")

	sb.WriteString("\n" + style.ColorDim(strings.Repeat("─", 40)) + "\n")
	sb.WriteString("[Enter/y] Confirm  [Esc/n] Cancel\n")

	return dialogStyle.Render(sb.String())
}
