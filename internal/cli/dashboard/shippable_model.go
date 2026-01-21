package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/shippable"
	"stackit.dev/stackit/internal/tui/core"
)

const (
	// autoRefreshInterval is the time between automatic refreshes
	autoRefreshInterval = 30 * time.Second
	// tickInterval is the interval for timer updates (for countdown display)
	tickInterval = 1 * time.Second
)

// dashboardState represents the current UI state of the dashboard.
type dashboardState int

const (
	stateLoading dashboardState = iota
	stateMain
	stateAnalyzing
	stateConfirming
	stateShipping
	statePublishing
	stateHelp
)

// ShippableOptions defines configuration for the shippable dashboard.
type ShippableOptions struct {
	// RunLocalCI enables local CI validation during combination checks.
	RunLocalCI bool
}

// shippableModel is the main model for the shippable work dashboard.
type shippableModel struct {
	core.BaseModel // Provides Ready signaling, spinner, and common message handling

	ctx    *app.Context
	engine engine.Engine
	cfg    config.Configurer

	// Analysis state
	analyzer    *shippable.Analyzer
	combiner    *shippable.Combiner
	analysis    *shippable.AnalysisResult
	combination *shippable.CombinationResult

	// UI state
	state         dashboardState // Current UI state (replaces boolean flags)
	stacks        []shippable.Stack
	selectedIndex int
	expanded      map[string]bool  // Tracks which stacks are expanded
	selected      map[string]bool  // Tracks which stacks are selected for shipping
	focusedStack  *shippable.Stack // Currently focused stack for detail view

	// Status
	lastRefresh   time.Time
	statusMessage string
	errorMessage  string

	// Progress tracking
	progress        progress.Model
	progressStep    int
	progressTotal   int
	progressMessage string

	// Confirmation action (used when state == stateConfirming)
	confirmAction string

	// Options
	options ShippableOptions
}

// keyMap defines all keyboard shortcuts for the dashboard.
type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Select    key.Binding
	Expand    key.Binding
	Ship      key.Binding
	Publish   key.Binding
	Analyze   key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding
	Confirm   key.Binding
	Cancel    key.Binding
	SelectAll key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle selection"),
	),
	Expand: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "expand/collapse"),
	),
	Ship: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "ship selected"),
	),
	Publish: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "restack & submit all"),
	),
	Analyze: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "analyze combination"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter", "y"),
		key.WithHelp("enter/y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "n"),
		key.WithHelp("esc/n", "cancel"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "select all shippable"),
	),
}

// newShippableModel creates a new shippable dashboard model.
func newShippableModel(ctx *app.Context, cfg config.Configurer, opts ShippableOptions) *shippableModel {
	analyzer := shippable.NewAnalyzer(ctx.Engine, ctx.GitHubClient)
	combiner := shippable.NewCombiner(ctx.Engine, cfg, ctx.Output)

	// Create progress bar
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return &shippableModel{
		ctx:      ctx,
		engine:   ctx.Engine,
		cfg:      cfg,
		analyzer: analyzer,
		combiner: combiner,
		expanded: make(map[string]bool),
		selected: make(map[string]bool),
		state:    stateLoading,
		options:  opts,
		progress: p,
	}
}

// selectedStacks returns the stacks that are currently selected.
func (m *shippableModel) selectedStacks() []shippable.Stack {
	var result []shippable.Stack
	for _, s := range m.stacks {
		if m.selected[s.RootBranch()] {
			result = append(result, s)
		}
	}
	return result
}

// selectedCount returns the number of selected stacks.
func (m *shippableModel) selectedCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

// currentStack returns the currently focused stack, or nil if none.
func (m *shippableModel) currentStack() *shippable.Stack {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.stacks) {
		return &m.stacks[m.selectedIndex]
	}
	return nil
}

// toggleSelection toggles selection of the current stack.
func (m *shippableModel) toggleSelection() {
	stack := m.currentStack()
	if stack == nil {
		return
	}
	root := stack.RootBranch()
	m.selected[root] = !m.selected[root]
}

// toggleExpand toggles expansion of the current stack.
func (m *shippableModel) toggleExpand() {
	stack := m.currentStack()
	if stack == nil {
		return
	}
	root := stack.RootBranch()
	m.expanded[root] = !m.expanded[root]
}

// selectAllShippable selects all shippable stacks.
func (m *shippableModel) selectAllShippable() {
	for _, s := range m.stacks {
		if s.IsShippable() {
			m.selected[s.RootBranch()] = true
		}
	}
}

// clearSelection clears all selections.
func (m *shippableModel) clearSelection() {
	m.selected = make(map[string]bool)
}

// Init initializes the model.
func (m *shippableModel) Init() tea.Cmd {
	m.SignalReady()
	return tea.Batch(m.InitSpinner(), m.refresh(), m.tick())
}

// tick returns a command that sends a tick message after the tick interval.
func (m *shippableModel) tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// HandleSpinnerMsg handles spinner tick messages for loading animations.
// Returns (handled, cmd) - if handled is true, the caller should return cmd.
func (m *shippableModel) HandleSpinnerMsg(msg tea.Msg) (bool, tea.Cmd) {
	// Use the BaseModel's spinner handling
	return m.BaseModel.HandleCommonMsg(msg)
}

// Messages for async operations
type (
	analysisCompleteMsg struct {
		result *shippable.AnalysisResult
		err    error
	}

	combinationCompleteMsg struct {
		result *shippable.CombinationResult
		err    error
	}

	refreshCompleteMsg struct {
		result *shippable.AnalysisResult
		err    error
	}

	publishCompleteMsg struct {
		restacked int
		submitted int
		err       error
	}

	// progressUpdateMsg updates the progress bar during async operations
	progressUpdateMsg struct {
		step    int
		total   int
		message string
	}

	// tickMsg is sent every second for countdown updates and auto-refresh
	tickMsg time.Time
)
