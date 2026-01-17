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
