// Package sync provides a TUI component for displaying sync progress.
package sync

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/tui/core"
	"stackit.dev/stackit/internal/tui/style"
)

// Phase represents a sync phase
type Phase string

// Phase constants
const (
	PhaseTrunk   Phase = "trunk"
	PhaseGitHub  Phase = "github"
	PhaseClean   Phase = "clean"
	PhaseRestack Phase = "restack"
)

// PhaseItem represents progress for a single phase
type PhaseItem struct {
	Phase   Phase
	Status  core.Status
	Message string
	Details []string // Lines of detail output for this phase
}

// Model is the bubbletea model for sync progress.
// It embeds core.BaseModel for standard lifecycle handling.
type Model struct {
	core.BaseModel // Embedded for ReadySignaler interface
	Phases         []PhaseItem
	CurrentPhase   Phase
	TotalOps       int
	CompletedOps   int
	Progress       progress.Model
	spinner        spinner.Model // Use local spinner for custom style
	Summary        string
}

// PhaseStartMsg indicates a phase has started
type PhaseStartMsg struct {
	Phase Phase
}

// PhaseDetailMsg adds a detail line to a phase
type PhaseDetailMsg struct {
	Phase   Phase
	Message string
}

// ProgressTickMsg updates the progress bar
type ProgressTickMsg struct {
	Completed int
	Total     int
}

// CompleteMsg indicates sync is complete
type CompleteMsg struct {
	Summary string
}

// NewModel creates a new sync model
func NewModel(totalOps int) *Model {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	commonStyles := style.DefaultCommonStyles()
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = commonStyles.Spinner

	m := &Model{
		Phases: []PhaseItem{
			{Phase: PhaseTrunk, Status: core.StatusPending, Message: "📥 Pulling from remote..."},
			{Phase: PhaseGitHub, Status: core.StatusPending, Message: "🔄 Fetching PR info from GitHub..."},
			{Phase: PhaseClean, Status: core.StatusPending, Message: "🧹 Cleaning branches..."},
			{Phase: PhaseRestack, Status: core.StatusPending, Message: "📚 Restacking branches..."},
		},
		TotalOps: totalOps,
		Progress: p,
		spinner:  s,
	}
	m.Width = 80 // Set BaseModel's Width
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	// Signal that the program is ready to receive messages via BaseModel
	m.SignalReady()
	return m.spinner.Tick
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle spinner ticks with our custom spinner BEFORE HandleCommonMsg
	// (HandleCommonMsg would update BaseModel.Spinner instead)
	if tickMsg, ok := msg.(spinner.TickMsg); ok {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(tickMsg)
		return m, cmd
	}

	// Handle common messages via BaseModel (key events, window resize)
	if handled, cmd := m.HandleCommonMsg(msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// BaseModel already set Width/Height, but we also need to update Progress.Width
		newWidth := msg.Width - 10
		if newWidth > 60 {
			newWidth = 60
		}
		m.Progress.Width = newWidth
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd

	case PhaseStartMsg:
		m.CurrentPhase = msg.Phase
		for i := range m.Phases {
			if m.Phases[i].Phase == msg.Phase {
				m.Phases[i].Status = core.StatusActive
			} else if m.Phases[i].Status == core.StatusActive {
				m.Phases[i].Status = core.StatusDone
			}
		}
		return m, nil

	case PhaseDetailMsg:
		for i := range m.Phases {
			if m.Phases[i].Phase == msg.Phase {
				m.Phases[i].Details = append(m.Phases[i].Details, msg.Message)
				break
			}
		}
		return m, nil

	case ProgressTickMsg:
		m.CompletedOps = msg.Completed
		m.TotalOps = msg.Total
		if m.TotalOps > 0 {
			cmd := m.Progress.SetPercent(float64(m.CompletedOps) / float64(m.TotalOps))
			return m, cmd
		}
		return m, nil

	case CompleteMsg:
		m.Done = true
		m.Summary = msg.Summary
		// Mark all phases as done
		for i := range m.Phases {
			if m.Phases[i].Status == core.StatusActive {
				m.Phases[i].Status = core.StatusDone
			}
		}
		return m, tea.Quit
	}

	return m, nil
}

// View renders the model
func (m *Model) View() string {
	var b strings.Builder

	// Get shared styles
	statusStyles := style.DefaultStatusStyles()
	commonStyles := style.DefaultCommonStyles()

	// Progress bar at top (only when active and not done)
	if !m.Done && m.TotalOps > 0 {
		b.WriteString(m.Progress.View())
		b.WriteString(fmt.Sprintf(" %d/%d\n\n", m.CompletedOps, m.TotalOps))
	}

	// Phase headers with their details
	firstPhase := true
	for _, phase := range m.Phases {
		if phase.Status == core.StatusPending {
			continue // Don't show pending phases
		}

		// Add blank line between phases (not before first)
		if !firstPhase {
			b.WriteString("\n")
		}
		firstPhase = false

		// Phase header
		icon := "✓"
		phaseStyle := statusStyles.Done
		if phase.Status == core.StatusActive {
			icon = m.spinner.View()
			phaseStyle = statusStyles.Active
		}

		b.WriteString(fmt.Sprintf("%s %s\n", icon, phaseStyle.Render(phase.Message)))

		// Phase details
		for _, detail := range phase.Details {
			b.WriteString(fmt.Sprintf("  %s\n", commonStyles.Subtle.Render(detail)))
		}
	}

	// Summary
	if m.Done && m.Summary != "" {
		b.WriteString("\n")
		b.WriteString(m.Summary)
		b.WriteString("\n")
	}

	return b.String()
}
