package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/shippable"
)

// Update handles all messages and updates the model.
func (m *shippableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshCompleteMsg:
		m.loading = false
		m.lastRefresh = time.Now()
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.errorMessage = ""
		m.analysis = msg.result
		m.stacks = msg.result.Stacks
		m.updateFocusedStack()
		return m, nil

	case analysisCompleteMsg:
		m.analyzing = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.analysis = msg.result
		m.stacks = msg.result.Stacks
		m.statusMessage = "Analysis complete"
		return m, nil

	case combinationCompleteMsg:
		m.analyzing = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.combination = msg.result
		if msg.result.Combinable {
			m.statusMessage = "Selected stacks can be combined"
		} else {
			m.statusMessage = "Some stacks conflict"
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m *shippableModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If showing confirmation, handle confirm/cancel
	if m.showConfirmation {
		return m.handleConfirmationKey(msg)
	}

	// If showing help, any key closes it
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// If loading or analyzing, only allow quit
	if m.loading || m.analyzing {
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}

	// Handle normal navigation and actions
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
			m.updateFocusedStack()
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		if m.selectedIndex < len(m.stacks)-1 {
			m.selectedIndex++
			m.updateFocusedStack()
		}
		return m, nil

	case key.Matches(msg, keys.Select):
		m.toggleSelection()
		return m, nil

	case key.Matches(msg, keys.Expand):
		m.toggleExpand()
		return m, nil

	case key.Matches(msg, keys.SelectAll):
		m.selectAllShippable()
		return m, nil

	case key.Matches(msg, keys.Ship):
		return m.startShip()

	case key.Matches(msg, keys.Analyze):
		return m.startCombinationAnalysis()

	case key.Matches(msg, keys.Refresh):
		m.loading = true
		m.statusMessage = "Refreshing..."
		return m, m.refresh()

	case key.Matches(msg, keys.Help):
		m.showHelp = true
		return m, nil
	}

	return m, nil
}

// handleConfirmationKey handles keyboard input during confirmation dialog.
func (m *shippableModel) handleConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Confirm):
		m.showConfirmation = false
		if m.confirmAction == "ship" {
			return m.executeShip()
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.showConfirmation = false
		m.confirmAction = ""
		return m, nil
	}
	return m, nil
}

// updateFocusedStack updates the focused stack based on selection.
func (m *shippableModel) updateFocusedStack() {
	m.focusedStack = m.currentStack()
}

// refresh fetches the latest stack analysis.
func (m *shippableModel) refresh() tea.Cmd {
	return func() tea.Msg {
		result, err := m.analyzer.AnalyzeAll(m.ctx.Context)
		return refreshCompleteMsg{result: result, err: err}
	}
}

// startShip initiates the ship workflow.
func (m *shippableModel) startShip() (tea.Model, tea.Cmd) {
	selected := m.selectedStacks()
	if len(selected) == 0 {
		m.errorMessage = "No stacks selected"
		return m, nil
	}

	// Check if all selected stacks are shippable
	for _, s := range selected {
		if !s.IsShippable() {
			m.errorMessage = "Cannot ship: " + s.RootBranch() + " is not shippable"
			return m, nil
		}
	}

	// Show confirmation dialog
	m.showConfirmation = true
	m.confirmAction = "ship"
	return m, nil
}

// executeShip executes the ship action after confirmation.
func (m *shippableModel) executeShip() (tea.Model, tea.Cmd) {
	// TODO: Implement actual ship action in Phase 4
	m.statusMessage = "Ship action not yet implemented"
	return m, nil
}

// startCombinationAnalysis checks if selected stacks can be combined.
func (m *shippableModel) startCombinationAnalysis() (tea.Model, tea.Cmd) {
	selected := m.selectedStacks()
	if len(selected) == 0 {
		m.errorMessage = "No stacks selected for analysis"
		return m, nil
	}

	m.analyzing = true
	m.statusMessage = "Analyzing combination..."

	return m, func() tea.Msg {
		result, err := m.combiner.CheckCombination(m.ctx.Context, selected, shippable.CheckCombinationOptions{
			RunLocalCI: m.options.RunLocalCI,
		})
		return combinationCompleteMsg{result: result, err: err}
	}
}
