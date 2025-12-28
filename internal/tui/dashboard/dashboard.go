// Package dashboard provides the interactive stack dashboard TUI.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kballard/go-shellquote"

	submitAction "stackit.dev/stackit/internal/actions/submit"
	syncAction "stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/operations"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
)

const (
	refreshInterval = 2 * time.Second
)

type tickMsg time.Time

type activityMsg struct {
	message string
	isError bool
}

// Options defines configuration and callbacks for the dashboard.
type Options struct {
	CommandFunc func(args []string) (string, error)
}

type model struct {
	ctx           *runtime.Context
	engine        engine.Engine
	renderer      *tree.StackTreeRenderer
	width         int
	height        int
	lastRefresh   time.Time
	currentBranch string

	options Options

	// Navigation
	branches       []string
	selectedIndex  int
	selectedBranch string

	// Pending Changes
	pendingChanges []engine.PendingChange

	// Activity Log
	activity []activityItem

	// Input
	commandInput textinput.Model

	// Operation state
	activeOperation   operations.Operation
	operationProgress []operations.Progress
	progressChan      <-chan operations.Progress

	// UI state
	showHelp bool

	err error
}

type activityItem struct {
	timestamp time.Time
	message   string
	isError   bool
}

// operationProgressMsg wraps a progress update from an operation
type operationProgressMsg operations.Progress

// operationDoneMsg signals an operation has completed
type operationDoneMsg struct{}

// Run starts the interactive dashboard program.
func Run(ctx *runtime.Context, opts Options) error {
	ti := textinput.New()
	ti.Placeholder = "Enter command (e.g. create -m \"msg\") or 'quit'"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 100

	m := &model{
		ctx:          ctx,
		engine:       ctx.Engine,
		options:      opts,
		commandInput: ti,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.refresh(),
		m.tick(),
	)
}

func (m *model) tick() tea.Cmd {
	return tea.Every(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) refresh() tea.Cmd {
	return func() tea.Msg {
		// Re-read engine state
		trunkName := m.engine.Trunk().GetName()
		if err := m.engine.Rebuild(trunkName); err != nil {
			return err
		}

		current := m.engine.CurrentBranch()
		currentName := ""
		if current != nil {
			currentName = current.GetName()
		}

		trunk := m.engine.Trunk().GetName()

		// Create renderer
		renderer := tree.NewStackTreeRenderer(
			currentName,
			trunk,
			func(name string) []string {
				children := m.engine.GetChildrenInternal(name)
				names := make([]string, len(children))
				for i, c := range children {
					names[i] = c.GetName()
				}
				return names
			},
			func(name string) string {
				parent := m.engine.GetParent(m.engine.GetBranch(name))
				if parent == nil {
					return ""
				}
				return parent.GetName()
			},
			func(name string) bool {
				return m.engine.IsTrunkInternal(name)
			},
			func(name string) bool {
				return m.engine.IsBranchUpToDateInternal(name)
			},
		)

		// Populate annotations
		allBranches := m.engine.AllBranches()
		for _, b := range allBranches {
			ann := tree.BranchAnnotation{}

			// PR Info
			if pr, _ := m.engine.GetPrInfo(b); pr != nil {
				ann.PRNumber = pr.Number()
				ann.PRState = pr.State()
				ann.IsDraft = pr.IsDraft()
			}

			// Stats
			added, deleted, _ := m.engine.GetDiffStatsInternal(b.GetName())
			ann.LinesAdded = added
			ann.LinesDeleted = deleted

			count, _ := m.engine.GetCommitCountInternal(b.GetName())
			ann.CommitCount = count

			// Scope
			ann.Scope = m.engine.GetScopeInternal(b.GetName()).String()
			ann.ExplicitScope = m.engine.GetExplicitScopeInternal(b.GetName()).String()

			renderer.SetAnnotation(b.GetName(), ann)
		}

		// Calculate flattened branch list for navigation
		branches := []string{trunk}
		var collect func(string)
		collect = func(name string) {
			children := m.engine.GetChildrenInternal(name)
			for _, child := range children {
				branches = append(branches, child.GetName())
				collect(child.GetName())
			}
		}
		collect(trunk)

		// Pending changes
		pending, _ := m.engine.GetPendingChanges(m.ctx.Context)

		return refreshMsg{
			renderer:       renderer,
			currentBranch:  currentName,
			branches:       branches,
			pendingChanges: pending,
		}
	}
}

type refreshMsg struct {
	renderer       *tree.StackTreeRenderer
	currentBranch  string
	branches       []string
	pendingChanges []engine.PendingChange
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If help is showing, any key closes it
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// If operation is running, only allow cancel
		if m.activeOperation != nil {
			if msg.String() == "esc" || msg.String() == "ctrl+c" {
				m.activeOperation.Cancel()
				m.activeOperation = nil
				m.operationProgress = nil
				m.progressChan = nil
				return m, m.refresh()
			}
			return m, nil
		}

		// If input has focus (has content), handle input keys
		if m.commandInput.Value() != "" {
			switch msg.String() {
			case "enter":
				val := m.commandInput.Value()
				m.commandInput.SetValue("")
				if val == "quit" || val == "exit" {
					return m, tea.Quit
				}
				return m, m.runCommand(val)
			case "esc":
				m.commandInput.SetValue("")
				return m, nil
			default:
				var cmd tea.Cmd
				m.commandInput, cmd = m.commandInput.Update(msg)
				return m, cmd
			}
		}

		// Handle keyboard shortcuts when input is empty
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		case "enter":
			return m, m.checkout()
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.selectedBranch = m.branches[m.selectedIndex]
			}
			return m, nil
		case "down", "j":
			if m.selectedIndex < len(m.branches)-1 {
				m.selectedIndex++
				m.selectedBranch = m.branches[m.selectedIndex]
			}
			return m, nil
		case "?":
			m.showHelp = true
			return m, nil
		case "s":
			return m, m.startSubmit(false)
		case "S":
			return m, m.startSubmit(true)
		case "y":
			return m, m.startSync()
		case "r":
			return m, m.runCommand("restack")
		}

		// Pass other keys to input
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(m.refresh(), m.tick())

	case refreshMsg:
		m.renderer = msg.renderer
		m.currentBranch = msg.currentBranch
		m.branches = msg.branches
		m.pendingChanges = msg.pendingChanges

		// Initialize selection if not set
		if m.selectedBranch == "" {
			m.selectedBranch = m.currentBranch
			if m.selectedBranch == "" && len(m.branches) > 0 {
				m.selectedBranch = m.branches[0]
			}
		}

		// Update selectedIndex based on selectedBranch
		found := false
		for i, b := range m.branches {
			if b == m.selectedBranch {
				m.selectedIndex = i
				found = true
				break
			}
		}
		if !found && len(m.branches) > 0 {
			m.selectedIndex = 0
			m.selectedBranch = m.branches[0]
		}

		m.lastRefresh = time.Now()

	case activityMsg:
		m.activity = append(m.activity, activityItem{
			timestamp: time.Now(),
			message:   msg.message,
			isError:   msg.isError,
		})
		if len(m.activity) > 5 {
			m.activity = m.activity[1:]
		}

	case operationProgressMsg:
		progress := operations.Progress(msg)
		m.operationProgress = append(m.operationProgress, progress)

		// Check if operation is done
		switch progress.Status {
		case operations.StatusCompleted:
			m.activity = append(m.activity, activityItem{
				timestamp: time.Now(),
				message:   progress.Step,
			})
			m.activeOperation = nil
			m.operationProgress = nil
			m.progressChan = nil
			return m, m.refresh()

		case operations.StatusFailed:
			errMsg := progress.Step
			if progress.Error != nil {
				errMsg = progress.Error.Error()
			}
			m.activity = append(m.activity, activityItem{
				timestamp: time.Now(),
				message:   errMsg,
				isError:   true,
			})
			m.activeOperation = nil
			m.operationProgress = nil
			m.progressChan = nil
			return m, m.refresh()

		case operations.StatusCanceled:
			m.activity = append(m.activity, activityItem{
				timestamp: time.Now(),
				message:   "Operation canceled",
			})
			m.activeOperation = nil
			m.operationProgress = nil
			m.progressChan = nil
			return m, m.refresh()
		}

		// Keep listening for more progress
		return m, m.waitForProgress()

	case operationDoneMsg:
		m.activeOperation = nil
		m.operationProgress = nil
		m.progressChan = nil
		return m, m.refresh()

	case error:
		m.err = msg
	}

	return m, nil
}

func (m *model) checkout() tea.Cmd {
	return func() tea.Msg {
		if m.selectedBranch == "" || m.selectedBranch == m.currentBranch {
			return nil
		}

		branch := m.engine.GetBranch(m.selectedBranch)
		if err := m.engine.CheckoutBranch(m.ctx.Context, branch); err != nil {
			return activityMsg{message: fmt.Sprintf("Failed to checkout %s: %v", m.selectedBranch, err), isError: true}
		}

		return tea.Batch(
			func() tea.Msg { return activityMsg{message: fmt.Sprintf("Checked out %s", m.selectedBranch)} },
			m.refresh(),
		)()
	}
}

func (m *model) runCommand(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		if m.options.CommandFunc == nil {
			return activityMsg{message: "Command execution not supported", isError: true}
		}

		args, err := shellquote.Split(cmdStr)
		if err != nil {
			return activityMsg{message: fmt.Sprintf("Invalid command: %v", err), isError: true}
		}

		if len(args) == 0 {
			return nil
		}

		m.ctx.Splog.SetQuiet(true)
		defer m.ctx.Splog.SetQuiet(false)

		output, err := m.options.CommandFunc(args)
		if err != nil {
			return activityMsg{message: fmt.Sprintf("Command failed: %v", err), isError: true}
		}

		msg := "Command complete"
		if output != "" {
			// Just show the first line of output if it's too long
			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) > 0 {
				msg = lines[0]
			}
		}

		return tea.Batch(
			func() tea.Msg { return activityMsg{message: msg} },
			m.refresh(),
		)()
	}
}

// startSubmit starts a submit operation
func (m *model) startSubmit(stack bool) tea.Cmd {
	return func() tea.Msg {
		opts := submitAction.Options{
			Stack:        stack,
			SubmitFooter: true,
		}

		op := operations.NewSubmitOperation(m.ctx, opts)
		m.activeOperation = op
		m.operationProgress = nil
		m.progressChan = op.Start(m.ctx.Context)

		// Return first progress wait
		return m.waitForProgressSync()
	}
}

// startSync starts a sync operation
func (m *model) startSync() tea.Cmd {
	return func() tea.Msg {
		opts := syncAction.Options{
			Restack: true,
		}

		op := operations.NewSyncOperation(m.ctx, opts)
		m.activeOperation = op
		m.operationProgress = nil
		m.progressChan = op.Start(m.ctx.Context)

		// Return first progress wait
		return m.waitForProgressSync()
	}
}

// waitForProgress returns a command that waits for the next progress message
func (m *model) waitForProgress() tea.Cmd {
	if m.progressChan == nil {
		return nil
	}
	return func() tea.Msg {
		return m.waitForProgressSync()
	}
}

// waitForProgressSync waits for progress synchronously (for use in commands)
func (m *model) waitForProgressSync() tea.Msg {
	if m.progressChan == nil {
		return operationDoneMsg{}
	}

	progress, ok := <-m.progressChan
	if !ok {
		return operationDoneMsg{}
	}
	return operationProgressMsg(progress)
}

func (m *model) View() string {
	if m.err != nil {
		return style.ColorRed(fmt.Sprintf("Error: %v", m.err))
	}

	if m.renderer == nil {
		return "Loading..."
	}

	// Show help overlay
	if m.showHelp {
		return m.renderHelpOverlay()
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true, false)

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		Padding(0, 1).
		Width(m.width / 2)

	detailsStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(m.width / 2)

	// Header
	headerText := fmt.Sprintf("Stackit Dashboard | Refresh: %s", m.lastRefresh.Format("15:04:05"))
	if m.activeOperation != nil {
		headerText = fmt.Sprintf("Stackit Dashboard | %s", m.getOperationStatus())
	}
	header := headerStyle.Width(m.width).Render(headerText)

	// Main content height
	contentHeight := m.height - 12

	// Left Pane: Stack Tree (or operation overlay)
	var treeContent string
	if m.activeOperation != nil {
		treeContent = m.renderOperationOverlay()
	} else {
		treeLines := m.renderer.RenderStack(m.engine.Trunk().GetName(), tree.RenderOptions{
			Short:          false,
			SelectedBranch: m.selectedBranch,
		})
		treeContent = strings.Join(treeLines, "\n")
	}

	// Right Pane: Branch Details
	detailsContent := m.renderDetails()

	// Combine panes
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		paneStyle.Height(contentHeight).Render(treeContent),
		detailsStyle.Height(contentHeight).Render(detailsContent),
	)

	// Status Line (Pending Changes Summary)
	statusLine := m.renderStatusLine()

	// Activity Log
	activityLines := []string{lipgloss.NewStyle().Bold(true).Render("Recent Activity:")}
	for _, item := range m.activity {
		msg := fmt.Sprintf("[%s] %s", item.timestamp.Format("15:04:05"), item.message)
		if item.isError {
			msg = style.ColorRed(msg)
		} else {
			msg = style.ColorDim(msg)
		}
		activityLines = append(activityLines, msg)
	}
	for len(activityLines) < 4 {
		activityLines = append(activityLines, "")
	}
	activity := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(0, 1).
		Width(m.width).
		Render(strings.Join(activityLines, "\n"))

	// Command Bar
	commandBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Padding(0, 1).
		Render("> " + m.commandInput.View())

	// Help line
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1).
		Width(m.width)
	var helpText string
	if m.activeOperation != nil {
		helpText = "esc: cancel operation"
	} else {
		helpText = "s: submit | S: submit stack | y: sync | r: restack | ?: help | q: quit"
	}
	help := helpStyle.Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		panes,
		statusLine,
		activity,
		commandBar,
		help,
	)
}

// getOperationStatus returns a status string for the current operation
func (m *model) getOperationStatus() string {
	if len(m.operationProgress) == 0 {
		return "Starting..."
	}
	last := m.operationProgress[len(m.operationProgress)-1]
	return last.Step
}

// renderOperationOverlay renders the operation progress overlay
func (m *model) renderOperationOverlay() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	stepStyle := lipgloss.NewStyle()

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Operation in Progress") + "\n\n")

	for _, p := range m.operationProgress {
		var icon string
		switch p.Status {
		case operations.StatusCompleted:
			icon = style.ColorCyan("✓")
		case operations.StatusFailed:
			icon = style.ColorRed("✗")
		case operations.StatusRunning:
			icon = style.ColorYellow("⏳")
		case operations.StatusSkipped:
			icon = style.ColorDim("○")
		default:
			icon = " "
		}

		line := fmt.Sprintf("%s %s", icon, p.Step)
		if p.Branch != "" {
			line = fmt.Sprintf("%s %s: %s", icon, p.Branch, p.Step)
		}
		sb.WriteString(stepStyle.Render(line) + "\n")
	}

	if len(m.operationProgress) > 0 {
		last := m.operationProgress[len(m.operationProgress)-1]
		if last.Total > 0 {
			sb.WriteString("\n")
			sb.WriteString(progressStyle.Render(fmt.Sprintf("Progress: %d/%d", last.Current, last.Total)))
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(style.ColorDim("Press Esc to cancel"))

	return sb.String()
}

// renderHelpOverlay renders the help panel
func (m *model) renderHelpOverlay() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Width(12)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Stackit Dashboard Help") + "\n\n")

	sb.WriteString(sectionStyle.Render("Navigation") + "\n")
	sb.WriteString(keyStyle.Render("j/k, ↑/↓") + descStyle.Render("Move selection up/down") + "\n")
	sb.WriteString(keyStyle.Render("enter") + descStyle.Render("Checkout selected branch") + "\n")

	sb.WriteString(sectionStyle.Render("Operations") + "\n")
	sb.WriteString(keyStyle.Render("s") + descStyle.Render("Submit current branch") + "\n")
	sb.WriteString(keyStyle.Render("S") + descStyle.Render("Submit entire stack") + "\n")
	sb.WriteString(keyStyle.Render("y") + descStyle.Render("Sync (pull trunk, restack)") + "\n")
	sb.WriteString(keyStyle.Render("r") + descStyle.Render("Restack branches") + "\n")

	sb.WriteString(sectionStyle.Render("Commands") + "\n")
	sb.WriteString(descStyle.Render("Type any stackit command and press enter:") + "\n")
	sb.WriteString(style.ColorDim("  create -m \"message\"  - Create new branch") + "\n")
	sb.WriteString(style.ColorDim("  modify               - Amend current commit") + "\n")
	sb.WriteString(style.ColorDim("  merge                - Merge approved PRs") + "\n")

	sb.WriteString(sectionStyle.Render("Other") + "\n")
	sb.WriteString(keyStyle.Render("?") + descStyle.Render("Toggle this help") + "\n")
	sb.WriteString(keyStyle.Render("q, quit") + descStyle.Render("Exit dashboard") + "\n")
	sb.WriteString(keyStyle.Render("esc") + descStyle.Render("Clear input / cancel operation") + "\n")

	sb.WriteString("\n" + style.ColorDim("Press any key to close"))

	return lipgloss.NewStyle().Padding(2, 4).Render(sb.String())
}

func (m *model) renderStatusLine() string {
	if len(m.pendingChanges) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1).Render("No pending changes")
	}

	staged := 0
	unstaged := 0
	for _, c := range m.pendingChanges {
		if c.Staged {
			staged++
		} else {
			unstaged++
		}
	}

	parts := []string{}
	if staged > 0 {
		parts = append(parts, style.ColorCyan(fmt.Sprintf("%d staged", staged)))
	}
	if unstaged > 0 {
		parts = append(parts, style.ColorYellow(fmt.Sprintf("%d unstaged", unstaged)))
	}

	return lipgloss.NewStyle().Padding(0, 1).Render("Changes: " + strings.Join(parts, ", "))
}

func (m *model) renderDetails() string {
	if m.selectedBranch == "" {
		return "No branch selected"
	}

	b := m.engine.GetBranch(m.selectedBranch)
	ann := m.renderer.Annotations[m.selectedBranch]

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Underline(true).MarginBottom(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Branch: "+m.selectedBranch) + "\n\n")

	// Basic Info
	sb.WriteString(labelStyle.Render("Status:") + " " + m.getStatusString(m.selectedBranch) + "\n")
	sb.WriteString(labelStyle.Render("Commits:") + " " + valueStyle.Render(fmt.Sprintf("%d", ann.CommitCount)) + "\n")
	sb.WriteString(labelStyle.Render("Changes:") + " " + style.ColorCyan(fmt.Sprintf("+%d", ann.LinesAdded)) + " " + style.ColorRed(fmt.Sprintf("-%d", ann.LinesDeleted)) + "\n")

	scope := m.engine.GetScopeInternal(m.selectedBranch)
	sb.WriteString(labelStyle.Render("Scope:") + " " + style.ColorScope(scope.String()) + "\n")

	parent := m.engine.GetParent(b)
	if parent != nil {
		sb.WriteString(labelStyle.Render("Parent:") + " " + valueStyle.Render(parent.GetName()) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("Pull Request") + "\n")
	if ann.PRNumber != nil {
		sb.WriteString(labelStyle.Render("Number:") + " " + style.ColorPRNumber(*ann.PRNumber) + "\n")
		sb.WriteString(labelStyle.Render("State:") + " " + valueStyle.Render(ann.PRState) + "\n")
		if ann.IsDraft {
			sb.WriteString(labelStyle.Render("Draft:") + " " + style.ColorYellow("Yes") + "\n")
		}
	} else {
		sb.WriteString(style.ColorDim("No PR associated") + "\n")
	}

	// Show pending changes if this is the current branch
	if m.selectedBranch == m.currentBranch && len(m.pendingChanges) > 0 {
		sb.WriteString("\n")
		sb.WriteString(titleStyle.Render("Working Directory") + "\n")
		for i, c := range m.pendingChanges {
			if i >= 5 {
				sb.WriteString(style.ColorDim(fmt.Sprintf("... and %d more", len(m.pendingChanges)-5)) + "\n")
				break
			}
			status := c.Status
			if c.Staged {
				status = style.ColorCyan(status)
			} else {
				status = style.ColorYellow(status)
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", status, c.Path))
		}
	}

	return sb.String()
}

func (m *model) getStatusString(branchName string) string {
	if m.engine.IsTrunkInternal(branchName) {
		return style.ColorDim("Trunk")
	}
	if !m.engine.IsBranchUpToDateInternal(branchName) {
		return style.ColorNeedsRestack("Needs Restack")
	}
	return style.ColorCyan("Up to date")
}
