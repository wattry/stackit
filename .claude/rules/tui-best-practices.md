# TUI Best Practices

## Architecture

### Handler Pattern
Separate business logic from UI via event handlers. Actions receive a `Handler` interface; callers provide implementations.

```go
// In actions/foo/handler.go
type Handler interface {
    Start(count int)
    OnStep(step Step, status StepStatus, message string)
    Complete(result Result)
    Cleanup()
    IsInteractive() bool
}

// NullHandler for when nil is passed
type NullHandler struct{}
```

### Simple + Interactive Handler Pairs
Always provide two implementations:
- `SimpleHandler` - text output for non-TTY (embed `common.BaseHandler`)
- `InteractiveHandler` - TUI for TTY (embeds SimpleHandler + Runner + Model)

```go
func NewFooUI(out output.Output, logger output.Logger) (*tui.Runner, Handler) {
    if tui.IsTTY() {
        model := component.NewModel()
        runner := tui.NewRunner(model, out, logger)
        runner.Start()
        return runner, NewInteractiveHandler(out, runner, model)
    }
    return nil, NewSimpleHandler(out)
}
```

### Runner Lifecycle
Runner manages terminal state, signals, panic recovery. Caller must `defer runner.Cleanup()`.

```go
runner, handler := NewFooUI(ctx.Output, ctx.Logger)
if runner != nil {
    defer runner.Cleanup()
}
```

## Model Structure

### BaseModel Embedding
Embed `core.BaseModel` for standard lifecycle handling:

```go
type Model struct {
    core.BaseModel  // ReadySignaler, Done, Width, Height, Spinner
    // ... component fields
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

### Message Types
Define typed messages for all communication. Never use raw strings or generic types.

```go
type PhaseStartMsg struct{ Phase Phase }
type ValidationProgressMsg struct {
    Current int
    Total   int
    Branch  string
}
type CompleteMsg struct{ Summary string }
```

## Styling

### Use Centralized Styles
Import from `tui/style` for consistency:

```go
statusStyles := style.DefaultStatusStyles()
statusIcons := style.DefaultStatusIcons()
commonStyles := style.DefaultCommonStyles()

icon := statusIcons.Done
text := statusStyles.Done.Render("Complete")
subtle := commonStyles.Subtle.Render("details")
```

### Status Types
Use `core.Status` for unified state tracking:
- `StatusPending` - not started
- `StatusActive` - in progress
- `StatusDone` - completed successfully
- `StatusError` - failed
- `StatusSkipped` - bypassed

## Terminal Management

### Pause/Resume for Prompts
Release terminal before prompts, restore after:

```go
func (h *InteractiveHandler) PromptConfirm(preview Preview) (bool, error) {
    if h.runner != nil {
        h.runner.Pause()
    }

    // Show preview and prompt...
    confirmed, err := tui.PromptConfirm("Proceed?", true)

    if confirmed && h.runner != nil {
        h.runner.Resume()
    }
    return confirmed, nil
}
```

### Window Size Handling
Always handle `tea.WindowSizeMsg`:

```go
case tea.WindowSizeMsg:
    m.Progress.Width = min(msg.Width-10, 60)
    return m, nil
```

## Error Handling

### Panic Recovery
Use `tui.SafeCmd` for commands that perform IO:

```go
cmd := tui.SafeCmd("fetch-data", logger, func() tea.Msg {
    // IO operation that might panic
})
```

### Timeout Protection
Use `SendWithTimeout` or `MustSend` for critical messages:

```go
runner.MustSend(msg)  // 5s timeout, logs on failure
```

## Key Patterns

- Actions never import TUI packages; they receive Handler interfaces
- Models use value receivers for Update/View (immutability)
- Only show phases that have started (hide pending)
- Progress bars: use `bubbles/progress` with `WithoutPercentage()` when showing explicit counts
- Batch commands with `tea.Batch()` when multiple are needed
