# TUI Development Guide

## Key Files

- `internal/tui/core/base_model.go` - BaseModel, ReadySignaler, key constants
- `internal/tui/style/colors.go` - Color constants, theme detection
- `internal/tui/style/theme.go` - Style structures (HeaderStyles, LayoutStyles, etc.)
- `internal/tui/components/tree/tree.go` - Stack tree renderer
- `internal/tui/runner.go` - Runner, SafeCmd, Send utilities

## Architecture

### Handler Pattern

Actions use a Handler interface to separate business logic from UI:

```
internal/actions/<action>/handler.go    - Defines Handler interface
internal/cli/stack/<action>_handlers.go - Implements handlers
```

Handler interfaces define:
- Callbacks for UI events (Start, OnStep, Complete, Cleanup)
- Prompts (PromptConfirm, PromptRename, etc.)
- `IsInteractive() bool` to check mode
- `NullHandler` struct for non-interactive/testing use

### Handler Implementations

Provide two implementations:
- `SimpleHandler` - text output for non-TTY (embed `cli/common.BaseHandler`)
- `InteractiveHandler` - TUI for TTY

```go
func NewFooUI(out output.Output, logger output.Logger) (*tui.Runner, Handler) {
    if tui.IsTTY() {
        model := NewModel()
        runner := tui.NewRunner(model, out, logger)
        runner.Start()
        return runner, NewInteractiveHandler(out, runner, model)
    }
    return nil, NewSimpleHandler(out)
}

// Caller must cleanup
runner, handler := NewFooUI(ctx.Output, ctx.Logger)
if runner != nil {
    defer runner.Cleanup()
}
```

## Models

### BaseModel

Embed `core.BaseModel` for standard lifecycle handling. Import from `tui/core` to avoid import cycles.

```go
type Model struct {
    core.BaseModel  // Spinner, Done, Width, Height
    // ... fields
}

func (m *Model) Init() tea.Cmd {
    m.SignalReady()  // MUST call to prevent Send() race conditions
    return m.InitSpinner()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if handled, cmd := m.HandleCommonMsg(msg); handled {
        return m, cmd
    }
    // Handle component-specific messages...
}
```

### Key Constants

Use constants from `tui/core` instead of string literals:

```go
core.KeyCtrlC  // "ctrl+c"
core.KeyEnter  // "enter"
core.KeyEsc    // "esc"
core.KeyQuit   // "q"
core.KeyUp     // "up"
core.KeyDown   // "down"
```

### Status Types

Use `core.Status` for state tracking:

```go
core.StatusPending  // not started
core.StatusActive   // in progress
core.StatusDone     // completed successfully
core.StatusError    // failed
core.StatusSkipped  // bypassed
core.StatusWaiting  // waiting on dependency
```

### Message Types

Define typed messages for communication. Never use raw strings.

```go
type PhaseStartMsg struct{ Phase Phase }
type CompleteMsg struct{ Summary string }
```

## Styling

Import from `tui/style`. Never use inline `lipgloss.NewStyle()` with magic color numbers.

```go
headerStyles := style.DefaultHeaderStyles()
headerStyles.Title.Render("Select Branch")

layoutStyles := style.DefaultLayoutStyles()
layoutStyles.Container.Render(content)  // standard margin (1, 2)

selectionStyles := style.DefaultSelectionStyles()
selectionStyles.Highlighted.Render("> " + item)

statusStyles := style.DefaultStatusStyles()
statusStyles.Done.Render("Complete")
```

### Color Constants

```go
style.ColorPrimary   // 205 - magenta, titles/emphasis
style.ColorSuccess   // 42  - green, done states
style.ColorError     // 196 - red, error states
style.ColorWarning   // 214 - orange, warnings
style.ColorInsert    // 10  - bright green, new items
style.ColorPending   // 240 - dark gray, pending
```

## Tree Rendering

Always use `tree.StackTreeRenderer` for branch visualizations. Never implement custom tree rendering.

```go
renderer := tree.NewRenderer(data)
renderer.SetAnnotation(branchName, tree.BranchAnnotation{
    CustomLabel: "(marker)",
})
lines := renderer.RenderStack(trunk, tree.RenderOptions{
    HideSummary: true,           // hide PR info for previews
    SkipSelectionPrefix: true,   // omit cursor padding
})
```

For virtual trees (e.g., showing where a branch will be inserted), implement `tree.Data` interface. See `virtualDirectionTree` in `direction_select.go`.

## Terminal Management

### Pause/Resume for Prompts

```go
func (h *InteractiveHandler) PromptConfirm(preview Preview) (bool, error) {
    if h.runner != nil {
        h.runner.Pause()
        defer h.runner.Resume()
    }
    return tui.PromptConfirm("Proceed?", true)
}
```

### Error Handling

```go
// Panic recovery for IO operations
cmd := tui.SafeCmd("fetch-data", logger, func() tea.Msg {
    return doFetch()
})

// Timeout protection for critical messages
runner.MustSend(msg)  // 5s timeout, logs on failure
```

## Patterns

### Multi-Step Flows

Use a single model with explicit state machine, not multiple bubbletea programs:

```go
type flowState int
const (
    stateSelecting flowState = iota
    stateConfirming
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state {
    case stateSelecting:
        return m.updateSelecting(msg)
    case stateConfirming:
        return m.updateConfirming(msg)
    }
    return m, nil
}
```

### Async Operations

Use `tea.Cmd` for async work, return results as typed messages:

```go
func (m Model) validate() tea.Cmd {
    return func() tea.Msg {
        return validationResultMsg{doValidation()}
    }
}
```

### Preview -> Confirm -> Execute

1. Build preview data
2. Call `handler.PromptConfirm*(preview)`
3. If confirmed, execute
4. Call `handler.Complete(result)`

For operations that might fail, run validation early and include results in the preview.

### Live Validation

Pass `ValidateSelection` callback to `LogOptions`. The LogModel handles debouncing and displays results in the footer.
