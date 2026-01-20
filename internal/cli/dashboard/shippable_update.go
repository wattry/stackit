package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/shippable"
)

// Update handles all messages and updates the model.
func (m *shippableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size messages (we don't use HandleCommonMsg because we have custom quit handling)
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.Width = wsMsg.Width
		m.Height = wsMsg.Height
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case refreshCompleteMsg:
		m.state = stateMain
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
		m.state = stateMain
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.analysis = msg.result
		m.stacks = msg.result.Stacks
		m.statusMessage = "Analysis complete"
		return m, nil

	case combinationCompleteMsg:
		m.state = stateMain
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

	case shipCompleteMsg:
		m.state = stateMain
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		// Ship successful
		m.statusMessage = "Shipped! PR: " + msg.result.PRURL
		m.clearSelection()
		return m, m.refresh()
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m *shippableModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If showing confirmation, handle confirm/cancel
	if m.state == stateConfirming {
		return m.handleConfirmationKey(msg)
	}

	// If showing help, any key closes it
	if m.state == stateHelp {
		m.state = stateMain
		return m, nil
	}

	// If loading, analyzing, or shipping, only allow quit
	if m.state == stateLoading || m.state == stateAnalyzing || m.state == stateShipping {
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
		// Clear previous combination result when selection changes
		m.combination = nil
		// Auto-analyze if multiple stacks selected
		if m.selectedCount() >= 2 {
			return m.startCombinationAnalysis()
		}
		return m, nil

	case key.Matches(msg, keys.Expand):
		m.toggleExpand()
		return m, nil

	case key.Matches(msg, keys.SelectAll):
		m.selectAllShippable()
		// Clear previous combination result when selection changes
		m.combination = nil
		// Auto-analyze if multiple stacks selected
		if m.selectedCount() >= 2 {
			return m.startCombinationAnalysis()
		}
		return m, nil

	case key.Matches(msg, keys.Ship):
		return m.startShip()

	case key.Matches(msg, keys.Analyze):
		return m.startCombinationAnalysis()

	case key.Matches(msg, keys.Refresh):
		m.state = stateLoading
		m.statusMessage = "Refreshing..."
		return m, m.refresh()

	case key.Matches(msg, keys.Help):
		m.state = stateHelp
		return m, nil
	}

	return m, nil
}

// handleConfirmationKey handles keyboard input during confirmation dialog.
func (m *shippableModel) handleConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Confirm):
		m.state = stateMain
		if m.confirmAction == "ship" {
			return m.executeShip()
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.state = stateMain
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
	m.state = stateConfirming
	m.confirmAction = "ship"
	return m, nil
}

// shipCompleteMsg signals that the ship operation completed.
type shipCompleteMsg struct {
	result *shippable.ShipResult
	err    error
}

// executeShip executes the ship action after confirmation.
func (m *shippableModel) executeShip() (tea.Model, tea.Cmd) {
	selected := m.selectedStacks()
	if len(selected) == 0 {
		m.errorMessage = "No stacks selected"
		return m, nil
	}

	m.state = stateShipping
	m.statusMessage = "Shipping..."

	return m, func() tea.Msg {
		shipper := shippable.NewShipper(m.ctx)
		result, err := shipper.Ship(selected, shippable.ShipOptions{
			SkipLocalCI: !m.options.RunLocalCI,
			Wait:        false, // Don't wait in UI, user can monitor PR
		})
		return shipCompleteMsg{result: result, err: err}
	}
}

// startCombinationAnalysis checks if selected stacks can be combined.
func (m *shippableModel) startCombinationAnalysis() (tea.Model, tea.Cmd) {
	selected := m.selectedStacks()
	if len(selected) == 0 {
		m.errorMessage = "No stacks selected for analysis"
		return m, nil
	}

	m.state = stateAnalyzing
	m.statusMessage = "Analyzing combination..."

	return m, func() tea.Msg {
		result, err := m.combiner.CheckCombination(m.ctx.Context, selected, shippable.CheckCombinationOptions{
			RunLocalCI: m.options.RunLocalCI,
		})
		return combinationCompleteMsg{result: result, err: err}
	}
}
