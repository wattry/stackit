package sync

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/core"
)

func TestNewModel(t *testing.T) {
	model := NewModel(10)

	assert.Equal(t, 10, model.TotalOps)
	assert.Equal(t, 0, model.CompletedOps)
	assert.False(t, model.Done)
	assert.Equal(t, 5, len(model.Phases)) // trunk, branches, github, clean, restack

	// All phases should be pending
	for _, phase := range model.Phases {
		assert.Equal(t, core.StatusPending, phase.Status)
	}
}

func TestModel_Init(t *testing.T) {
	model := NewModel(0)

	// Set up ready channel to capture signal
	readyChan := make(chan struct{})
	model.SetReadyChan(readyChan)

	cmd := model.Init()

	// Should signal ready
	select {
	case <-readyChan:
		// Ready signal received as expected
	default:
		t.Error("expected ready signal but channel was not closed")
	}

	// Should return a spinner tick command
	require.NotNil(t, cmd, "Init should return a command for spinner tick")
}

func TestModel_Update_PhaseStartMsg(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Start trunk phase
	newModel, _ := model.Update(PhaseStartMsg{Phase: PhaseTrunk})
	m := newModel.(*Model)

	assert.Equal(t, PhaseTrunk, m.CurrentPhase)

	// Find trunk phase and verify it's active
	for _, phase := range m.Phases {
		if phase.Phase == PhaseTrunk {
			assert.Equal(t, core.StatusActive, phase.Status)
		} else {
			assert.Equal(t, core.StatusPending, phase.Status)
		}
	}
}

func TestModel_Update_PhaseTransition(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Start trunk phase
	newModel, _ := model.Update(PhaseStartMsg{Phase: PhaseTrunk})
	m := newModel.(*Model)

	// Start github phase - trunk should become done
	newModel, _ = m.Update(PhaseStartMsg{Phase: PhaseGitHub})
	m = newModel.(*Model)

	assert.Equal(t, PhaseGitHub, m.CurrentPhase)

	// Verify phase statuses
	for _, phase := range m.Phases {
		switch phase.Phase {
		case PhaseTrunk:
			assert.Equal(t, core.StatusDone, phase.Status, "trunk should be done")
		case PhaseGitHub:
			assert.Equal(t, core.StatusActive, phase.Status, "github should be active")
		default:
			assert.Equal(t, core.StatusPending, phase.Status, "%s should be pending", phase.Phase)
		}
	}
}

func TestModel_Update_PhaseDetailMsg(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Start trunk phase first
	newModel, _ := model.Update(PhaseStartMsg{Phase: PhaseTrunk})
	m := newModel.(*Model)

	// Add a detail to trunk phase
	newModel, _ = m.Update(PhaseDetailMsg{
		Phase:   PhaseTrunk,
		Message: "main fast-forwarded to abc1234",
	})
	m = newModel.(*Model)

	// Find trunk phase and verify detail was added
	for _, phase := range m.Phases {
		if phase.Phase == PhaseTrunk {
			require.Len(t, phase.Details, 1)
			assert.Equal(t, "main fast-forwarded to abc1234", phase.Details[0])
		}
	}
}

func TestModel_Update_ProgressTickMsg(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Update progress
	newModel, cmd := model.Update(ProgressTickMsg{
		Completed: 5,
		Total:     10,
	})
	m := newModel.(*Model)

	assert.Equal(t, 5, m.CompletedOps)
	assert.Equal(t, 10, m.TotalOps)
	assert.NotNil(t, cmd, "should return progress update command")
}

func TestModel_Update_CompleteMsg(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Send complete message
	newModel, cmd := model.Update(CompleteMsg{
		Summary: "All done!",
	})
	m := newModel.(*Model)

	assert.True(t, m.Done)
	assert.Equal(t, "All done!", m.Summary)

	// Should return quit command
	require.NotNil(t, cmd)
}

func TestModel_Update_KeyMsg_Quit(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"ctrl+c quits", "ctrl+c"},
		{"q quits", "q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel(10)
			model.Init()

			_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			// For ctrl+c, use the proper key type
			if tt.key == "ctrl+c" {
				_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			}

			assert.NotNil(t, cmd, "should return quit command")
		})
	}
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	model := NewModel(10)
	model.Init()

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := newModel.(*Model)

	assert.Equal(t, 100, m.Width)
	assert.Equal(t, 50, m.Height)
	// Progress width should be capped at 60
	assert.Equal(t, 60, m.Progress.Width)
}

func TestModel_Update_WindowSizeMsg_NarrowTerminal(t *testing.T) {
	model := NewModel(10)
	model.Init()

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 50, Height: 30})
	m := newModel.(*Model)

	assert.Equal(t, 50, m.Width)
	assert.Equal(t, 30, m.Height)
	// Progress width should be 40 (50 - 10)
	assert.Equal(t, 40, m.Progress.Width)
}

func TestModel_View_InProgress(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Start a phase
	newModel, _ := model.Update(PhaseStartMsg{Phase: PhaseTrunk})
	m := newModel.(*Model)

	// Update progress
	newModel, _ = m.Update(ProgressTickMsg{Completed: 3, Total: 10})
	m = newModel.(*Model)

	view := m.View()

	// Should contain progress info
	assert.Contains(t, view, "3/10")
}

func TestModel_View_Completed(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Complete the operation
	newModel, _ := model.Update(CompleteMsg{Summary: "Everything synced!"})
	m := newModel.(*Model)

	view := m.View()

	// Should contain summary
	assert.Contains(t, view, "Everything synced!")
}

func TestModel_Update_SpinnerTickMsg(t *testing.T) {
	model := NewModel(10)
	model.Init()

	// Send a spinner tick
	_, cmd := model.Update(spinner.TickMsg{})

	// Should return another tick command to keep spinner animating
	assert.NotNil(t, cmd, "should return spinner tick command")
}

// TestMessageRecorder_Usage demonstrates how MessageRecorder can be used
// to test message flow in more complex scenarios
func TestMessageRecorder_Usage(t *testing.T) {
	recorder := tui.NewMessageRecorder()

	// Record some messages
	recorder.Record(PhaseStartMsg{Phase: PhaseTrunk})
	recorder.Record(PhaseDetailMsg{Phase: PhaseTrunk, Message: "test"})
	recorder.Record(ProgressTickMsg{Completed: 1, Total: 5})

	assert.Equal(t, 3, recorder.Count())

	// Check for specific message types
	hasPhaseStart := recorder.HasMessage(func(msg tea.Msg) bool {
		_, ok := msg.(PhaseStartMsg)
		return ok
	})
	assert.True(t, hasPhaseStart)

	hasComplete := recorder.HasMessage(func(msg tea.Msg) bool {
		_, ok := msg.(CompleteMsg)
		return ok
	})
	assert.False(t, hasComplete)

	// Reset and verify
	recorder.Reset()
	assert.Equal(t, 0, recorder.Count())
}
