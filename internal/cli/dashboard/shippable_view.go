package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/shippable"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
)

// View renders the dashboard.
func (m *shippableModel) View() string {
	switch m.state {
	case stateLoading:
		return m.renderLoading()
	case stateHelp:
		return m.renderHelp()
	case stateConfirming:
		return m.renderConfirmation()
	default:
		return m.renderMain()
	}
}

// renderLoading shows the loading state.
func (m *shippableModel) renderLoading() string {
	return lipgloss.NewStyle().
		Padding(2, 4).
		Render("Loading shippable work...")
}

// renderMain renders the main dashboard view.
func (m *shippableModel) renderMain() string {
	// Calculate dimensions
	headerHeight := 3
	footerHeight := 4
	cartHeight := 0
	if m.selectedCount() > 0 {
		cartHeight = 8 // Fixed height for cart panel
	}

	contentHeight := m.Height - headerHeight - footerHeight - cartHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	leftWidth := m.Width * 2 / 3
	rightWidth := m.Width - leftWidth

	// Build sections
	header := m.renderHeader()
	leftPane := m.renderStackList(leftWidth, contentHeight)
	rightPane := m.renderDetailsPanel(rightWidth, contentHeight)
	footer := m.renderFooter()

	// Combine stack viewer panes (top row)
	stackViewer := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Build final layout
	var sections []string
	sections = append(sections, header, stackViewer)

	// Add cart panel if items selected (bottom row)
	if m.selectedCount() > 0 {
		cartPane := m.renderCartPanel(m.Width, cartHeight)
		sections = append(sections, cartPane)
	}

	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the dashboard header.
func (m *shippableModel) renderHeader() string {
	title := titleStyle.Render("SHIPPABLE WORK")

	var status string
	switch {
	case m.state == stateAnalyzing || m.state == stateShipping:
		status = headerStatusStyle.Render("Analyzing...")
	case m.errorMessage != "":
		status = errorTextStyle.Render(m.errorMessage)
	case m.statusMessage != "":
		status = headerStatusStyle.Render(m.statusMessage)
	case m.analysis != nil:
		status = headerStatusStyle.Render(fmt.Sprintf("%d stacks (%d shippable)",
			m.analysis.TotalStacks(), m.analysis.ShippableCount))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Width(m.Width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, title, "  ", status))
}

// renderStackList renders the list of stacks.
func (m *shippableModel) renderStackList(width, height int) string {
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		Padding(0, 1).
		Width(width).
		Height(height)

	if len(m.stacks) == 0 {
		return paneStyle.Render(paneHeaderStyle.Render("STACKS") + "\n\n" +
			commonStyles.Dim.Render("No stacks found. Create one with `stackit create`"))
	}

	var sb strings.Builder
	sb.WriteString(paneHeaderStyle.Render("STACKS") + "\n")
	sb.WriteString(strings.Repeat("─", width-4) + "\n")

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

	return paneStyle.Render(sb.String())
}

// renderStackLine renders a single stack in the list.
func (m *shippableModel) renderStackLine(stack shippable.Stack, selected bool) string {
	// Selection checkbox
	var checkbox string
	if m.selected[stack.RootBranch()] {
		checkbox = style.ColorCyan("[x]")
	} else {
		checkbox = "[ ]"
	}

	// Status icon
	statusIcon := m.getStatusIcon(stack.Status)

	// Stack name and branch count
	name := stack.RootBranch()
	branchCount := fmt.Sprintf("(%d branches)", stack.BranchCount())

	// Expand indicator
	expandIndicator := "▶"
	if m.expanded[stack.RootBranch()] {
		expandIndicator = "▼"
	}

	// Build line
	line := fmt.Sprintf("%s %s %s %s %s", checkbox, statusIcon, name, branchCount, expandIndicator)

	// Highlight if selected
	if selected {
		line = selectedRowStyle.Render(line)
	}

	return line
}

// renderBranchLine renders a single branch within an expanded stack.
func (m *shippableModel) renderBranchLine(branchName string) string {
	branch := m.engine.GetBranch(branchName)
	if branch.GetName() == "" {
		return style.ColorDim("├── " + branchName)
	}

	// Get PR info if available
	prStatus, err := branch.GetPRSubmissionStatus()
	var prInfo string
	if err == nil && prStatus.PRNumber != nil && *prStatus.PRNumber > 0 {
		prInfo = fmt.Sprintf(" #%d", *prStatus.PRNumber)
	}

	return style.ColorDim("├── ") + branchName + style.ColorDim(prInfo)
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
	paneStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(width).
		Height(height)

	var sb strings.Builder
	sb.WriteString(paneHeaderStyle.Render("DETAILS") + "\n")
	sb.WriteString(strings.Repeat("─", width-4) + "\n")

	if m.focusedStack != nil {
		sb.WriteString(m.renderStackDetails(m.focusedStack))
	} else {
		sb.WriteString(commonStyles.Dim.Render("Select a stack to see details"))
	}

	return paneStyle.Render(sb.String())
}

// renderCartPanel renders the bottom cart panel when stacks are selected.
func (m *shippableModel) renderCartPanel(width, height int) string {
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(0, 1).
		Width(width).
		Height(height)

	selected := m.selectedStacks()
	totalBranches := 0
	for _, s := range selected {
		totalBranches += s.BranchCount()
	}

	// Cart header with count badge
	cartHeader := paneHeaderStyle.Render("SHIPPING")
	countBadge := lipgloss.NewStyle().
		Background(lipgloss.Color("6")).
		Foreground(lipgloss.Color("0")).
		Padding(0, 1).
		Render(fmt.Sprintf("%d stacks", len(selected)))

	// Build cart items inline
	itemParts := make([]string, 0, len(selected))
	for _, s := range selected {
		statusIcon := m.getStatusIcon(s.Status)
		itemParts = append(itemParts, fmt.Sprintf("%s %s", statusIcon, s.RootBranch()))
	}
	items := strings.Join(itemParts, "  •  ")

	// Combination status (compact)
	var analysisStatus string
	if m.combination != nil {
		if m.combination.Combinable {
			analysisStatus = style.ColorGreen("✓ Compatible")
		} else {
			analysisStatus = style.ColorRed("✗ Conflicts")
		}
	} else {
		analysisStatus = style.ColorDim("[a] analyze")
	}

	// Ship button
	canShip := true
	for _, s := range selected {
		if !s.IsShippable() {
			canShip = false
			break
		}
	}

	var shipAction string
	if canShip {
		shipAction = buttonPrimary.Render("[s] Ship")
	} else {
		shipAction = buttonDisabled.Render("[s] Ship") + " " + style.ColorDim("(not ready)")
	}

	// Build the panel content as a horizontal layout
	content := fmt.Sprintf("%s %s    %s    %s (%d branches)    %s",
		cartHeader, countBadge, items, analysisStatus, totalBranches, shipAction)

	return paneStyle.Render(content)
}

// renderStackDetails renders detailed info about a stack with tree view.
func (m *shippableModel) renderStackDetails(stack *shippable.Stack) string {
	var sb strings.Builder

	// Header with status badge
	statusBadge := m.renderStatusBadge(stack.Status)
	sb.WriteString(commonStyles.Bold.Render(stack.RootBranch()) + " " + statusBadge + "\n")

	// Quick stats row
	statsRow := m.renderQuickStats(stack)
	sb.WriteString(statsRow + "\n\n")

	// Stack tree visualization
	sb.WriteString(style.ColorDim("Stack:") + "\n")
	treeLines := m.renderStackTree(stack)
	for _, line := range treeLines {
		sb.WriteString(line + "\n")
	}

	// Blocking PRs (if any)
	if len(stack.BlockingPRs) > 0 {
		sb.WriteString("\n" + style.ColorYellow("Blocking:") + "\n")
		for _, bp := range stack.BlockingPRs {
			reason := m.formatBlockingReason(bp.Reason)
			sb.WriteString(fmt.Sprintf("  %s %s\n", style.ColorDim("•"), bp.Branch))
			sb.WriteString(fmt.Sprintf("    %s\n", reason))
		}
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
	// Create a filter that only includes branches in this stack
	stackBranches := make(map[string]bool)
	for _, branch := range stack.Stack.AllBranches {
		stackBranches[branch] = true
	}

	filter := func(branchName string) bool {
		return stackBranches[branchName]
	}

	// Create tree renderer with filter
	renderer := tui.NewStackTreeRendererWithFilter(m.engine, filter)

	// Add annotations for each branch
	for _, branchName := range stack.Stack.AllBranches {
		branch := m.engine.GetBranch(branchName)
		if branch.GetName() == "" {
			continue
		}
		ann := tui.GetBranchAnnotation(m.engine, branch)

		// Add blocking info as custom label
		for _, bp := range stack.BlockingPRs {
			if bp.Branch == branchName {
				ann.CustomLabel = m.getBlockingIcon(bp.Reason)
				break
			}
		}

		renderer.SetAnnotation(branchName, ann)
	}

	// Render tree in compact mode
	opts := tree.RenderOptions{
		Mode:                tree.RenderModeCompact,
		HideSummary:         false,
		SkipSelectionPrefix: true,
	}

	return renderer.RenderStack(stack.RootBranch(), opts)
}

// getBlockingIcon returns an icon for a blocking reason.
func (m *shippableModel) getBlockingIcon(reason shippable.BlockingReason) string {
	switch reason {
	case shippable.ReasonCIFailing:
		return style.ColorRed("✗")
	case shippable.ReasonCIPending:
		return style.ColorYellow("⏳")
	case shippable.ReasonChangesRequested:
		return style.ColorRed("✗")
	case shippable.ReasonReviewRequired:
		return style.ColorYellow("○")
	case shippable.ReasonDraft:
		return style.ColorDim("Draft")
	case shippable.ReasonNoPR:
		return style.ColorDim("No PR")
	default:
		return ""
	}
}

// formatBlockingReason returns a human-readable blocking reason.
func (m *shippableModel) formatBlockingReason(reason shippable.BlockingReason) string {
	switch reason {
	case shippable.ReasonCIFailing:
		return style.ColorRed("CI checks failing")
	case shippable.ReasonCIPending:
		return style.ColorYellow("CI checks pending")
	case shippable.ReasonChangesRequested:
		return style.ColorRed("Changes requested")
	case shippable.ReasonReviewRequired:
		return style.ColorYellow("Review required")
	case shippable.ReasonDraft:
		return style.ColorDim("PR is a draft")
	case shippable.ReasonNoPR:
		return style.ColorDim("No PR created")
	default:
		return string(reason)
	}
}

// renderFooter renders the footer with keyboard shortcuts.
func (m *shippableModel) renderFooter() string {
	shortcuts := []string{
		"[Space] Toggle",
		"[Enter] Expand",
		"[j/k] Navigate",
		"[s] Ship",
		"[a] Analyze",
		"[r] Refresh",
		"[?] Help",
		"[q] Quit",
	}

	return footerStyle.Width(m.Width).Render(strings.Join(shortcuts, "  "))
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
	sb.WriteString(helpKeyStyle.Render("a") + helpDescStyle.Render("Analyze combination") + "\n")
	sb.WriteString(helpKeyStyle.Render("r") + helpDescStyle.Render("Refresh analysis") + "\n")

	sb.WriteString(helpSectionStyle.Render("Other") + "\n")
	sb.WriteString(helpKeyStyle.Render("?") + helpDescStyle.Render("Toggle this help") + "\n")
	sb.WriteString(helpKeyStyle.Render("q") + helpDescStyle.Render("Quit") + "\n")

	sb.WriteString("\n" + commonStyles.Dim.Render("Press any key to close"))

	return lipgloss.NewStyle().Padding(2, 4).Render(sb.String())
}

// renderConfirmation renders the ship confirmation dialog.
func (m *shippableModel) renderConfirmation() string {
	selected := m.selectedStacks()
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

	sb.WriteString("\n" + strings.Repeat("─", 40) + "\n")
	sb.WriteString("[Enter/y] Confirm  [Esc/n] Cancel\n")

	return lipgloss.NewStyle().Padding(2, 4).Render(sb.String())
}
