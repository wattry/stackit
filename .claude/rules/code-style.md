# Code Style Rules

## General

- Clarity and simplicity over cleverness
- No backwards compatibility concerns unless specified
- Do not leave code unimplemented, it is better to leave a TODO

## Go Code Style

- Prefer early returns over deep nesting
- Use meaningful variable names; single-letter names only for loop indices
- Remove unused function parameters entirely; don't use `_` to ignore them
- Avoid redundant CLI flags that duplicate positional arguments
- Use `switch` statements instead of if-else chains with 3+ conditions
- For boolean conditions, use `switch { case cond1: ... case cond2: ... }`

## Error Handling

- Always handle errors explicitly; never ignore them with `_`
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return errors to callers rather than logging and continuing

## Comments

- Write comments for exported functions and types
- Explain "why" not "what"
- Only add comments where code is not self-documenting

## Testing

- Use table-driven tests for multiple cases
- Keep test setup minimal and focused
- Don't add code after assertions that terminate the test (e.g., `if err != nil` after `require.NoError`)
- New CLI commands need integration tests in `internal/integration/`, not just unit tests
- Integration tests should use `NewTestShellInProcess(t)` for faster execution

## TUI Key Constants

Always use existing key constants from `internal/tui/tui.go`:
- `KeyCtrlC`, `KeyEsc`, `KeyQuit`, `KeyEnter`

Never use string literals like `"ctrl+c"` - the linter enforces constants for repeated strings.

## Type Naming

Before creating a new type in a package, check if a similar type already exists:
```bash
rg "type.*KeyMap" internal/tui/   # before creating key map types
rg "type.*Config" internal/tui/   # before creating config types
```

Prefix types with context to avoid conflicts (e.g., `moveConfirmKeys` not `confirmKeyMap`).
