# TUI Architecture

## Handler Pattern

Actions that need UI interaction use a Handler interface pattern:

```
internal/actions/<action>/handler.go   - Defines Handler interface
internal/cli/stack/<action>_handlers.go - Implements handlers
```

### Structure

1. **Handler interface** (`internal/actions/X/handler.go`):
   - Defines callbacks for UI events (Start, OnStep, Complete, etc.)
   - Defines prompts (PromptConfirm, PromptRename, etc.)
   - Includes `IsInteractive() bool` to check mode
   - Provides `NullHandler` for non-interactive/testing use

2. **Handler implementations** (`internal/cli/stack/X_handlers.go`):
   - `SimpleHandler` - Non-interactive, minimal output
   - `InteractiveHandler` - Full TUI prompts and feedback

### Example

```go
// internal/actions/move/handler.go
type Handler interface {
    Start(source, oldParent, newParent string)
    OnStep(step Step, status StepStatus, message string)
    Complete(result Result)
    IsInteractive() bool
    PromptConfirmMove(preview Preview) (bool, error)
}

// internal/cli/stack/move_handlers.go
type InteractiveMoveHandler struct { ... }
func (h *InteractiveMoveHandler) PromptConfirmMove(preview move.Preview) (bool, error) {
    // Show TUI preview, prompt user
}
```

## Tree Rendering System

### Always Use the Tree Component

**IMPORTANT:** When rendering a stack or branch tree visualization, always use `tree.StackTreeRenderer` from `internal/tui/components/tree`. Never implement custom tree rendering logic.

The tree component provides:
- Consistent visual appearance across all commands
- Proper handling of branch annotations, scopes, and status indicators
- Support for collapsed nodes, search filtering, and selection

**For virtual/modified trees** (e.g., showing where a new branch will be inserted), implement the `tree.Data` interface to provide the modified structure, then use `tree.NewRenderer()`. See `virtualDirectionTree` in `direction_select.go` for an example.

**Key options:**
- `HideSummary: true` - Hide stats/PR info for clean previews
- `SkipSelectionPrefix: true` - Omit selection cursor padding for previews

### Key Files

- `internal/tui/components/tree/tree.go` - Core `StackTreeRenderer` with `RenderOptions` and `BranchAnnotation`
- `internal/tui/tree_renderer.go` - Helper functions to create renderers from engine state
- `internal/tui/log_ui.go` - Interactive log/branch selector using the tree renderer

### Adding Visual Indicators

To add text/badges after branch names:
```go
annotation := tree.BranchAnnotation{
    CustomLabel: "(your label here)",
}
renderer.SetAnnotation(branchName, annotation)
```

### Making Branches Non-Selectable

To show branches in the tree but prevent selection (cursor skips them):
```go
opts := tui.LogOptions{
    NonSelectable: map[string]bool{
        "branch-to-skip": true,
    },
}
```

### Overriding Annotations

To add custom labels without rebuilding all annotations:
```go
opts := tui.LogOptions{
    AnnotationOverrides: map[string]tree.BranchAnnotation{
        "my-branch": {CustomLabel: "<---- marker"},
    },
}
```

## Common Patterns

### Preview → Confirm → Execute

Many actions follow this pattern:
1. Build preview data (what will happen)
2. Call `handler.PromptConfirm*(preview)`
3. If confirmed, execute the action
4. Call `handler.Complete(result)`

### Validation Before Preview

For operations that might fail (conflicts, etc.):
1. Run validation early
2. Include validation results in preview
3. Show user the preview WITH conflict info
4. Let them decide whether to proceed or cancel

## Bubbletea Idioms

### Multi-Step Interactive Flows

For flows with multiple steps (select → confirm → execute), use a **single unified model** with an explicit state machine, NOT multiple separate bubbletea programs.

```go
type flowState int
const (
    stateSelecting flowState = iota
    stateConfirming
    stateDone
)

type Model struct {
    state flowState
    // embed child models rather than calling them separately
    childModel *ChildModel
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state {
    case stateSelecting:
        return m.updateSelecting(msg)
    case stateConfirming:
        return m.updateConfirming(msg)
    }
    return m, nil
}

func (m Model) View() string {
    switch m.state {
    case stateSelecting:
        return m.childModel.View()
    case stateConfirming:
        return m.viewConfirmation()
    }
    return ""
}
```

### Async Operations via Commands

Use `tea.Cmd` for async work. Return results as custom message types:

```go
type validationResultMsg struct { result *Result }

func (m Model) validate() tea.Cmd {
    return func() tea.Msg {
        result := doValidation()
        return validationResultMsg{result}
    }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case validationResultMsg:
        m.result = msg.result
        return m, nil
    }
    // ...
}
```

### Composing Models

Embed child models and delegate to them:

```go
type ParentModel struct {
    child *ChildModel
}

func (m ParentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle parent-level keys first
    if keyMsg, ok := msg.(tea.KeyMsg); ok {
        if keyMsg.String() == "enter" {
            // Parent handles enter specially
            return m, nil
        }
    }

    // Delegate other messages to child
    updated, cmd := m.child.Update(msg)
    m.child = updated.(*ChildModel)
    return m, cmd
}
```

### Live Validation During Selection

When selection requires validation feedback (e.g., conflict checking), pass a `ValidateSelection` callback to `LogOptions`. The LogModel handles debouncing and displays results in the footer.
