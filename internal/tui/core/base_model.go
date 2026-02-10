// Package core provides foundational TUI types that can be imported without cycles.
// This package contains BaseModel, ReadySignaler, and key constants that components need.
package core

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Key constants for TUI interactions.
const (
	KeyCtrlC = "ctrl+c"
	KeyQuit  = "q"
	KeyEsc   = "esc"
	KeyEnter = "enter"
	KeyUp    = "up"
	KeyDown  = "down"
	KeyLeft  = "left"
	KeyRight = "right"
	KeyTab   = "tab"
)

// ReadySignaler allows a model to signal when it's ready to receive messages.
// Models implementing this interface will have their SetReadyChan called before
// the program starts, and should close the channel in their Init() method.
type ReadySignaler interface {
	SetReadyChan(chan struct{})
}

// BaseModel provides common functionality for all TUI models.
// Embed this in component models to get standard lifecycle handling.
//
// Usage:
//
//	type MyModel struct {
//	    core.BaseModel
//	    // ... other fields
//	}
//
//	func (m *MyModel) Init() tea.Cmd {
//	    m.SignalReady()
//	    return m.InitSpinner()
//	}
//
//	func (m *MyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	    if handled, cmd := m.HandleCommonMsg(msg); handled {
//	        return m, cmd
//	    }
//	    // ... handle other messages
//	}
type BaseModel struct {
	readyChan chan struct{}
	Spinner   spinner.Model
	Done      bool
	Width     int
	Height    int
}

// SetReadyChan implements ReadySignaler.
// This is called by the Runner before starting the program.
func (b *BaseModel) SetReadyChan(ch chan struct{}) {
	b.readyChan = ch
}

// SignalReady should be called from Init() to signal the runner that
// the model is ready to receive messages. This prevents race conditions
// where Send() is called before the event loop is running.
func (b *BaseModel) SignalReady() {
	if b.readyChan != nil {
		close(b.readyChan)
		b.readyChan = nil
	}
}

// HandleCommonMsg processes messages common to all models.
// Returns (handled bool, cmd tea.Cmd). If handled is true, the model
// should return the cmd and not process the message further.
//
// Handles:
// - tea.KeyPressMsg: "ctrl+c" and "q" quit the program
// - tea.WindowSizeMsg: updates Width and Height (returns handled=false so model can also handle)
// - spinner.TickMsg: updates the spinner
func (b *BaseModel) HandleCommonMsg(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == KeyCtrlC || msg.String() == KeyQuit {
			return true, tea.Quit
		}
	case tea.WindowSizeMsg:
		b.Width = msg.Width
		b.Height = msg.Height
		return false, nil // Let model also handle for custom logic
	case spinner.TickMsg:
		var cmd tea.Cmd
		b.Spinner, cmd = b.Spinner.Update(msg)
		return true, cmd
	}
	return false, nil
}

// InitSpinner initializes the spinner with default settings.
// Call this from Init() after SignalReady().
func (b *BaseModel) InitSpinner() tea.Cmd {
	b.Spinner = spinner.New()
	b.Spinner.Spinner = spinner.Dot
	b.Spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return b.Spinner.Tick
}

// MarkDone sets the Done flag to true.
// Call this when the operation is complete.
func (b *BaseModel) MarkDone() {
	b.Done = true
}

// IsDone returns true if the operation is complete.
func (b *BaseModel) IsDone() bool {
	return b.Done
}
