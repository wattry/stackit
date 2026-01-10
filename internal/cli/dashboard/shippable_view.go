package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/shippable"
	"stackit.dev/stackit/internal/tui/style"
)

// View renders the dashboard.
func (m *shippableModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	if m.showConfirmation {
		return m.renderConfirmation()
	}

	return m.renderMain()
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
	contentHeight := m.height - headerHeight - footerHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	leftWidth := m.width * 2 / 3
	rightWidth := m.width - leftWidth

	// Build sections
	header := m.renderHeader()
	leftPane := m.renderStackList(leftWidth, contentHeight)
	rightPane := m.renderDetailsPanel(rightWidth, contentHeight)
	footer := m.renderFooter()

	// Combine panes
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderHeader renders the dashboard header.
func (m *shippableModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	title := titleStyle.Render("SHIPPABLE WORK")

	var status string
	switch {
	case m.analyzing:
		status = statusStyle.Render("Analyzing...")
	case m.errorMessage != "":
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.errorMessage)
	case m.statusMessage != "":
		status = statusStyle.Render(m.statusMessage)
	case m.analysis != nil:
		status = statusStyle.Render(fmt.Sprintf("%d stacks (%d shippable)",
			m.analysis.TotalStacks(), m.analysis.ShippableCount))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, title, "  ", status))
}

// renderStackList renders the list of stacks.
func (m *shippableModel) renderStackList(width, height int) string {
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		Padding(0, 1).
		Width(width).
		Height(height)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	if len(m.stacks) == 0 {
		return paneStyle.Render(headerStyle.Render("STACKS") + "\n\n" +
			style.ColorDim("No stacks found. Create one with `stackit create`"))
	}

	var sb strings.Builder
	sb.WriteString(headerStyle.Render("STACKS") + "\n")
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
		line = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Render(line)
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

// renderDetailsPanel renders the right-side details panel.
func (m *shippableModel) renderDetailsPanel(width, height int) string {
	paneStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(width).
		Height(height)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	var sb strings.Builder

	// Selection summary
	selectedCount := m.selectedCount()
	if selectedCount > 0 {
		sb.WriteString(headerStyle.Render("SELECTION") + "\n")
		sb.WriteString(strings.Repeat("─", width-4) + "\n")
		sb.WriteString(fmt.Sprintf("Selected: %d stacks\n\n", selectedCount))

		for _, s := range m.selectedStacks() {
			sb.WriteString(fmt.Sprintf("  %s (%d branches)\n", s.RootBranch(), s.BranchCount()))
		}

		sb.WriteString("\n")

		// Combination status
		if m.combination != nil {
			if m.combination.Combinable {
				sb.WriteString(style.ColorGreen("Combinable: Yes") + "\n")
			} else {
				sb.WriteString(style.ColorRed("Combinable: No") + "\n")
				if len(m.combination.ConflictingStacks) > 0 {
					sb.WriteString("Conflicts:\n")
					for _, es := range m.combination.ConflictingStacks {
						sb.WriteString(fmt.Sprintf("  - %s\n", es.Stack.RootBranch()))
					}
				}
			}

			if m.combination.LocalCIPassed != nil {
				if *m.combination.LocalCIPassed {
					sb.WriteString(style.ColorGreen("Local CI: Passed") + "\n")
				} else {
					sb.WriteString(style.ColorRed("Local CI: Failed") + "\n")
				}
			} else {
				sb.WriteString(style.ColorDim("Local CI: Not run") + "\n")
			}
		}

		sb.WriteString("\n" + strings.Repeat("─", width-4) + "\n")
		sb.WriteString(style.ColorCyan("[S]") + " Ship selected\n")
		sb.WriteString(style.ColorCyan("[A]") + " Analyze combination\n")
	} else {
		sb.WriteString(headerStyle.Render("DETAILS") + "\n")
		sb.WriteString(strings.Repeat("─", width-4) + "\n")

		if m.focusedStack != nil {
			sb.WriteString(m.renderStackDetails(m.focusedStack))
		} else {
			sb.WriteString(style.ColorDim("Select a stack to see details"))
		}
	}

	return paneStyle.Render(sb.String())
}

// renderStackDetails renders detailed info about a stack.
func (m *shippableModel) renderStackDetails(stack *shippable.Stack) string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(12)

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(stack.RootBranch()) + "\n\n")

	// Status
	sb.WriteString(labelStyle.Render("Status:") + " " + m.getStatusLabel(stack.Status) + "\n")

	// Branch count
	sb.WriteString(labelStyle.Render("Branches:") + " " + fmt.Sprintf("%d", stack.BranchCount()) + "\n")

	// Approval status
	if stack.ApprovalOK {
		sb.WriteString(labelStyle.Render("Approval:") + " " + style.ColorGreen("Approved") + "\n")
	} else {
		sb.WriteString(labelStyle.Render("Approval:") + " " + style.ColorYellow("Pending") + "\n")
	}

	// CI status
	if stack.GitHubCIOK {
		sb.WriteString(labelStyle.Render("GitHub CI:") + " " + style.ColorGreen("Passing") + "\n")
	} else {
		sb.WriteString(labelStyle.Render("GitHub CI:") + " " + style.ColorRed("Failing/Pending") + "\n")
	}

	// Blocking PRs
	if len(stack.BlockingPRs) > 0 {
		sb.WriteString("\n" + lipgloss.NewStyle().Bold(true).Render("Blocking:") + "\n")
		for _, bp := range stack.BlockingPRs {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", bp.Branch, bp.Reason))
		}
	}

	return sb.String()
}

// getStatusLabel returns a styled label for a status.
func (m *shippableModel) getStatusLabel(status shippable.Status) string {
	switch status {
	case shippable.StatusShippable:
		return style.ColorGreen("Shippable")
	case shippable.StatusPending:
		return style.ColorYellow("Pending")
	case shippable.StatusBlocked:
		return style.ColorRed("Blocked")
	case shippable.StatusIncomplete:
		return style.ColorDim("Incomplete")
	default:
		return "Unknown"
	}
}

// renderFooter renders the footer with keyboard shortcuts.
func (m *shippableModel) renderFooter() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1).
		Width(m.width)

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

	return helpStyle.Render(strings.Join(shortcuts, "  "))
}

// renderHelp renders the help overlay.
func (m *shippableModel) renderHelp() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Width(12)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Shippable Work Dashboard Help") + "\n\n")

	sb.WriteString(sectionStyle.Render("Navigation") + "\n")
	sb.WriteString(keyStyle.Render("j/k, ↑/↓") + descStyle.Render("Move selection up/down") + "\n")
	sb.WriteString(keyStyle.Render("enter") + descStyle.Render("Expand/collapse stack") + "\n")

	sb.WriteString(sectionStyle.Render("Selection") + "\n")
	sb.WriteString(keyStyle.Render("space") + descStyle.Render("Toggle stack selection") + "\n")
	sb.WriteString(keyStyle.Render("A") + descStyle.Render("Select all shippable") + "\n")

	sb.WriteString(sectionStyle.Render("Actions") + "\n")
	sb.WriteString(keyStyle.Render("s") + descStyle.Render("Ship selected stacks") + "\n")
	sb.WriteString(keyStyle.Render("a") + descStyle.Render("Analyze combination") + "\n")
	sb.WriteString(keyStyle.Render("r") + descStyle.Render("Refresh analysis") + "\n")

	sb.WriteString(sectionStyle.Render("Other") + "\n")
	sb.WriteString(keyStyle.Render("?") + descStyle.Render("Toggle this help") + "\n")
	sb.WriteString(keyStyle.Render("q") + descStyle.Render("Quit") + "\n")

	sb.WriteString("\n" + style.ColorDim("Press any key to close"))

	return lipgloss.NewStyle().Padding(2, 4).Render(sb.String())
}

// renderConfirmation renders the ship confirmation dialog.
func (m *shippableModel) renderConfirmation() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginBottom(1)

	selected := m.selectedStacks()
	totalBranches := 0
	for _, s := range selected {
		totalBranches += s.BranchCount()
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("SHIP CONFIRMATION") + "\n\n")

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
