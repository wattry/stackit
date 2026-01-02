package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

const (
	logStyleFull = "FULL"
)

// LogMode defines how the log is used
type LogMode int

const (
	// LogModeView is the default view mode for browsing the log
	LogModeView LogMode = iota
	// LogModeSelect is the selection mode for choosing a branch
	LogModeSelect
)

// LogModel is the bubbletea model for the interactive log
type LogModel struct {
	context      context.Context
	engine       engine.Engine
	githubClient github.Client
	renderer     *tree.StackTreeRenderer
	viewport     viewport.Model
	width        int
	height       int
	ready        bool

	// State
	mode           LogMode
	branches       []tree.RenderedBranch // Visible branches with their lines
	selectedIndex  int
	selectedBranch string
	collapsed      map[string]bool
	canceled       bool

	// Options
	style         string
	reverse       bool
	showUntracked bool
	exclude       map[string]bool
}

// NewLogModel creates a new LogModel
func NewLogModel(ctx context.Context, eng engine.Engine, ghClient github.Client, opts LogOptions) *LogModel {
	m := &LogModel{
		context:       ctx,
		engine:        eng,
		githubClient:  ghClient,
		style:         opts.Style,
		reverse:       opts.Reverse,
		showUntracked: opts.ShowUntracked,
		exclude:       opts.Exclude,
		collapsed:     make(map[string]bool),
		mode:          LogModeView,
	}

	return m
}

// newLogSelectModel creates a new LogModel in selection mode
func newLogSelectModel(ctx context.Context, eng engine.Engine, ghClient github.Client, opts LogOptions) *LogModel {
	m := NewLogModel(ctx, eng, ghClient, opts)
	m.mode = LogModeSelect
	return m
}

// Init initializes the bubbletea model
func (m *LogModel) Init() tea.Cmd {
	return m.refresh()
}

func (m *LogModel) refresh() tea.Cmd {
	return func() tea.Msg {
		// Populate remote SHAs if needed
		if m.style == logStyleFull {
			_ = m.engine.PopulateRemoteShas()
		}

		// Prefetch CI status in batch if in FULL style
		var ciStatuses map[string]*github.CheckStatus
		allBranches := m.engine.AllBranches()
		if m.style == logStyleFull && m.githubClient != nil {
			branchNames := make([]string, 0, len(allBranches))
			for _, b := range allBranches {
				if !b.IsTrunk() {
					branchNames = append(branchNames, b.GetName())
				}
			}
			if len(branchNames) > 0 {
				ciStatuses, _ = m.githubClient.BatchGetPRChecksStatus(m.context, branchNames)
			}
		}

		// Collect annotations
		annotations := make(map[string]tree.BranchAnnotation)
		utils.Run(allBranches, func(b engine.Branch) {
			ann := GetBranchAnnotation(m.engine, b)
			// Add CI status if available
			if m.style == logStyleFull && !b.IsTrunk() && ciStatuses != nil {
				if status := ciStatuses[b.GetName()]; status != nil {
					ann.CheckStatus = tree.CheckStatusPassing
					if status.Pending {
						ann.CheckStatus = tree.CheckStatusPending
					} else if !status.Passing {
						ann.CheckStatus = tree.CheckStatusFailing
					}
				}
			}
			annotations[b.GetName()] = ann
		})

		return refreshLogMsg{annotations: annotations}
	}
}

type refreshLogMsg struct {
	annotations map[string]tree.BranchAnnotation
}

// Update handles message updates for the bubbletea model
func (m *LogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case KeyQuit, KeyCtrlC:
			m.canceled = true
			return m, tea.Quit
		case KeyEsc:
			if m.mode == LogModeSelect {
				m.canceled = true
				return m, tea.Quit
			}
			return m, tea.Quit
		case KeyUp, "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.selectedBranch = m.branches[m.selectedIndex].Name
				m.renderTree()
				m.ensureVisible()
			}
		case KeyDown, "j":
			if m.selectedIndex < len(m.branches)-1 {
				m.selectedIndex++
				m.selectedBranch = m.branches[m.selectedIndex].Name
				m.renderTree()
				m.ensureVisible()
			}
		case KeyEnter:
			if m.mode == LogModeSelect {
				return m, tea.Quit
			}
			if m.selectedBranch != "" {
				m.collapsed[m.selectedBranch] = !m.collapsed[m.selectedBranch]
				m.renderTree()
			}
		case " ":
			if m.selectedBranch != "" {
				m.collapsed[m.selectedBranch] = !m.collapsed[m.selectedBranch]
				m.renderTree()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
		m.renderTree()

	case refreshLogMsg:
		trunk := m.engine.Trunk()
		var filter func(string) bool
		if len(m.exclude) > 0 {
			filter = func(name string) bool {
				return !m.exclude[name]
			}
		}
		m.renderer = NewStackTreeRendererWithStrategy(m.engine, engine.SortStrategySmart, filter)
		m.renderer.SetAnnotations(msg.annotations)
		m.renderTree()

		// Initial selection
		if m.selectedBranch == "" {
			current := m.engine.CurrentBranch()
			if current != nil {
				m.selectedBranch = current.GetName()
			} else {
				m.selectedBranch = trunk.GetName()
			}
			// Find index
			for i, b := range m.branches {
				if b.Name == m.selectedBranch {
					m.selectedIndex = i
					break
				}
			}
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *LogModel) renderTree() {
	if m.renderer == nil || !m.ready {
		return
	}

	trunk := m.engine.Trunk().GetName()
	opts := tree.RenderOptions{
		Reverse:        m.reverse,
		SelectedBranch: m.selectedBranch,
		Collapsed:      m.collapsed,
		SingleLine:     m.mode == LogModeSelect,
	}
	m.branches = m.renderer.RenderStackDetailed(trunk, opts)

	var allLines []string
	for _, b := range m.branches {
		allLines = append(allLines, b.Lines...)
	}

	m.viewport.SetContent(strings.Join(allLines, "\n"))
}

func (m *LogModel) ensureVisible() {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.branches) {
		return
	}

	// Calculate the line offset for the selected branch
	lineOffset := 0
	for i := 0; i < m.selectedIndex; i++ {
		lineOffset += len(m.branches[i].Lines)
	}

	branchHeight := len(m.branches[m.selectedIndex].Lines)

	// Simple viewport scrolling to keep selected branch visible
	if lineOffset < m.viewport.YOffset {
		m.viewport.YOffset = lineOffset
	} else if lineOffset+branchHeight > m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = lineOffset + branchHeight - m.viewport.Height
	}
}

// View renders the bubbletea model
func (m *LogModel) View() string {
	if !m.ready || m.renderer == nil {
		return "Loading..."
	}

	title := "Stackit Log"
	help := "'q' to quit, 'enter' to expand/collapse, 'j/k' to scroll"
	if m.mode == LogModeSelect {
		title = "Select Branch"
		help = "'esc' to cancel, 'enter' to select, 'space' to expand/collapse, 'j/k' to scroll"
	}

	header := style.ColorDim(fmt.Sprintf(" %s | %d branches | %s", title, len(m.engine.AllBranches()), help))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.viewport.View(),
	)
}

// PromptLogSelect runs the interactive log in selection mode and returns the selected branch name
func PromptLogSelect(ctx context.Context, eng engine.Engine, ghClient github.Client, opts LogOptions) (string, error) {
	m := newLogSelectModel(ctx, eng, ghClient, opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	res := finalModel.(*LogModel)
	if res.canceled {
		return "", errors.ErrCanceled
	}

	return res.selectedBranch, nil
}

// LogOptions repeated here to avoid circular dependency if needed,
// but we'll probably use actions.LogOptions
type LogOptions struct {
	Style         string
	Reverse       bool
	ShowUntracked bool
	Exclude       map[string]bool // Branches to exclude from selection
}
