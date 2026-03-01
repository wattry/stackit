# Package Dependencies

The codebase follows a layered architecture. Avoid import cycles by respecting this hierarchy.

## Dependency Rules

```
internal/tui          → CAN import: engine, git, tui/components/*, tui/style
                      → CANNOT import: actions/*, cli/*

internal/actions/*    → CAN import: tui, engine, git, app, output
                      → CANNOT import: cli/*

internal/cli/*        → CAN import: actions/*, tui, engine, app
                      → This is the top layer, can import most things

internal/engine       → CAN import: git
                      → CANNOT import: tui, actions/*, cli/*

internal/git          → Lowest layer, minimal dependencies

apps/web/             → Consumes: internal/contracts/http (type contracts via API)
                      → CANNOT import: any Go packages directly

internal/contracts/http → Source of truth for API response shapes
internal/api/          → Serves web static assets + API endpoints
```

## Common Pitfalls

1. **TUI importing actions**: If `internal/tui` needs data structures from `internal/actions/X`, define a local struct in `tui` with the same fields and have the caller convert.

2. **Circular handler dependencies**: Action handlers are defined in `internal/actions/X/handler.go` but implemented in `internal/cli/stack/X_handlers.go`. The interface lives in actions, implementation in cli.

## Example: Avoiding Cycles

```go
// BAD - creates import cycle
// internal/tui/preview.go
import "stackit.dev/stackit/internal/actions/move"
func RenderPreview(p move.Preview) string { ... }

// GOOD - define local struct
// internal/tui/preview.go
type PreviewData struct {
    SourceBranch string
    // ... same fields
}
func RenderPreview(p PreviewData) string { ... }

// Caller in cli/ converts:
previewData := tui.PreviewData{
    SourceBranch: preview.SourceBranch,
    // ...
}
```
