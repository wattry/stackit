// Package tui provides terminal UI utilities.
package tui

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/output"
)

// ReadySignaler allows a model to signal when it's ready to receive messages.
// Models implementing this interface will have their SetReadyChan called before
// the program starts, and should close the channel in their Init() method.
type ReadySignaler interface {
	SetReadyChan(chan struct{})
}

// Runner manages async bubbletea program lifecycle with panic recovery.
// It handles signal handling, terminal cleanup, and crash logging.
type Runner struct {
	model   tea.Model
	output  output.Output
	logger  output.Logger
	program *tea.Program
	mu      sync.Mutex
	started bool
	stopped bool
	sigChan chan os.Signal
}

// NewRunner creates a new TUI runner for async program execution.
func NewRunner(model tea.Model, out output.Output, logger output.Logger) *Runner {
	return &Runner{
		model:  model,
		output: out,
		logger: logger,
	}
}

// Start begins running the tea.Program in a background goroutine.
// It sets up signal handling and panic recovery.
// If the model implements ReadySignaler, Start waits for the model to signal
// readiness before returning, preventing race conditions with Send().
func (r *Runner) Start() {
	startTime := time.Now()
	r.logger.Debug("tui.Runner.Start entering")

	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		r.logger.Debug("tui.Runner.Start already started, returning")
		return
	}
	r.started = true

	// Quiet console output while TUI is active
	r.output.SetQuiet(true)

	// Set up ready channel if model supports it
	var readyChan chan struct{}
	if signaler, ok := r.model.(ReadySignaler); ok {
		readyChan = make(chan struct{})
		signaler.SetReadyChan(readyChan)
		r.logger.Debug("tui.Runner.Start ready channel configured")
	}

	r.program = tea.NewProgram(r.model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Set up signal handler to ensure terminal is restored on interrupt
	r.sigChan = make(chan os.Signal, 1)
	signal.Notify(r.sigChan, os.Interrupt, syscall.SIGTERM)
	r.mu.Unlock()

	go func() {
		<-r.sigChan
		r.Cleanup()
		signal.Stop(r.sigChan)
	}()

	// Run program in background with panic recovery
	go func() {
		defer func() {
			if p := recover(); p != nil {
				stack := string(debug.Stack())
				r.logger.Error("TUI panic: %v\n%s", p, stack)
				// Print to stderr so user sees something
				fmt.Fprintf(os.Stderr, "\nstackit TUI crashed: %v\n", p)
				r.Cleanup()
			}
		}()

		if _, err := r.program.Run(); err != nil {
			r.logger.Error("TUI error: %v", err)
		}
		r.Cleanup()
	}()

	// Wait for ready signal if model supports it
	// This prevents Send() calls from blocking on an uninitialized event loop
	if readyChan != nil {
		select {
		case <-readyChan:
			r.logger.Debug("tui.Runner.Start ready signal received", "durationMs", time.Since(startTime).Milliseconds())
		case <-time.After(2 * time.Second):
			r.logger.Warn("tui.Runner.Start ready timeout, proceeding anyway", "durationMs", time.Since(startTime).Milliseconds())
		}
	}

	r.logger.Debug("tui.Runner.Start completed", "durationMs", time.Since(startTime).Milliseconds())
}

// Cleanup ensures the terminal is restored to normal mode.
// This is safe to call multiple times and on nil receivers.
func (r *Runner) Cleanup() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}

	p := r.program
	r.mu.Unlock()

	if p != nil {
		// Quit the program and wait for it to restore terminal state
		p.Quit()
		p.Wait()
	}

	r.mu.Lock()
	r.program = nil
	r.output.SetQuiet(false)
	r.stopped = true
	r.mu.Unlock()
}

// Pause releases the terminal for interactive prompts.
// This is safe to call on nil receivers.
func (r *Runner) Pause() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.program != nil {
		_ = r.program.ReleaseTerminal()
		r.output.SetQuiet(false)
	}
}

// Resume restores the TUI after Pause.
// This is safe to call on nil receivers.
func (r *Runner) Resume() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.program != nil {
		r.output.SetQuiet(true)
		_ = r.program.RestoreTerminal()
	}
}

// Send sends a message to the running program.
// Safe to call on nil receivers or when program is not running (no-op).
func (r *Runner) Send(msg tea.Msg) {
	if r == nil {
		return
	}

	msgType := reflect.TypeOf(msg).String()
	r.logger.Debug("tui.Runner.Send", "msgType", msgType)

	r.mu.Lock()
	p := r.program
	r.mu.Unlock()

	if p != nil {
		p.Send(msg)
	}
}

// Wait blocks until the program exits.
// Safe to call on nil receivers.
func (r *Runner) Wait() {
	if r == nil {
		return
	}
	r.mu.Lock()
	p := r.program
	r.mu.Unlock()

	if p != nil {
		p.Wait()
	}
}

// IsRunning returns true if the program is currently running.
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started && !r.stopped
}

// IsHealthy returns true if the TUI is running and responsive.
// This is a more comprehensive check than IsRunning, verifying the program is not nil.
func (r *Runner) IsHealthy() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started && !r.stopped && r.program != nil
}

// SendWithTimeout sends a message with a timeout.
// Returns error if the send doesn't complete within the timeout.
// This is useful for detecting hangs in the TUI event loop.
func (r *Runner) SendWithTimeout(msg tea.Msg, timeout time.Duration) error {
	if r == nil {
		return nil
	}

	msgType := reflect.TypeOf(msg).String()
	r.logger.Debug("tui.Runner.SendWithTimeout", "msgType", msgType, "timeout", timeout)

	done := make(chan struct{})
	go func() {
		r.Send(msg)
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		r.logger.Error("TUI send timed out", "msgType", msgType, "timeout", timeout)
		return fmt.Errorf("send timed out after %v for %s", timeout, msgType)
	}
}

// MustSend sends a message and logs error if timeout occurs.
// Uses a default 5-second timeout. This is the recommended way to send messages
// when you want hang detection without blocking indefinitely.
func (r *Runner) MustSend(msg tea.Msg) {
	if err := r.SendWithTimeout(msg, 5*time.Second); err != nil {
		r.logger.Error("MustSend failed", "error", err)
	}
}

// PanicError is sent when a tea.Cmd panics during execution.
// Models can handle this to show an error message or recover gracefully.
type PanicError struct {
	Source string // Name of the operation that panicked
	Err    error  // The panic value wrapped as an error
	Stack  string // Stack trace at the time of panic
}

func (p PanicError) Error() string {
	return fmt.Sprintf("%s panicked: %v", p.Source, p.Err)
}

// SafeCmd wraps a tea.Cmd with panic recovery.
// If the command panics, it logs the error and returns a PanicError message.
// This is useful for commands that perform IO or call external APIs.
func SafeCmd(name string, logger output.Logger, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		defer func() {
			if p := recover(); p != nil {
				stack := string(debug.Stack())
				err := fmt.Errorf("%v", p)
				if logger != nil {
					logger.Error("%s panicked: %v\n%s", name, p, stack)
				}
				// We can't return from inside defer, so we re-panic with a wrapped error
				// that will be caught by the outer recovery
				panic(PanicError{Source: name, Err: err, Stack: stack})
			}
		}()
		return cmd()
	}
}

// SafeCmdFunc wraps a function that returns tea.Msg with panic recovery.
// If the function panics, it logs the error and returns a PanicError message.
func SafeCmdFunc(name string, logger output.Logger, fn func() tea.Msg) tea.Cmd {
	return func() (result tea.Msg) {
		defer func() {
			if p := recover(); p != nil {
				stack := string(debug.Stack())
				err := fmt.Errorf("%v", p)
				if logger != nil {
					logger.Error("%s panicked: %v\n%s", name, p, stack)
				}
				result = PanicError{Source: name, Err: err, Stack: stack}
			}
		}()
		return fn()
	}
}
