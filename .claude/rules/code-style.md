# Code Style

## Go Patterns

- Early returns over deep nesting
- Meaningful names; single-letter only for loop indices
- Remove unused parameters entirely (don't use `_`)
- `switch` over if-else chains with 3+ conditions
- For boolean conditions: `switch { case cond1: ... case cond2: ... }`

## Error Handling

- Always handle errors explicitly (never `_`)
- Wrap with context: `fmt.Errorf("context: %w", err)`
- Return errors to callers; don't log and continue

## Testing

See `testing.md` for comprehensive testing guidelines. Key points:

- Table-driven tests for multiple cases
- Integration tests in `internal/integration/` using `NewTestShellInProcess(t)`
- Use `require` over `assert` for early failure
- Always use `t.Parallel()` for parallel test execution

## TUI

Use constants from `internal/tui/core/`:
```go
core.KeyCtrlC, core.KeyEsc, core.KeyQuit, core.KeyEnter
```

Never use string literals like `"ctrl+c"`.

Before creating types, check for existing ones:
```bash
rg "type.*KeyMap" internal/tui/
```

## General

- Clarity over cleverness
- Leave TODOs rather than unimplemented code
- No backwards compatibility unless specified
- Comments explain "why" not "what"
