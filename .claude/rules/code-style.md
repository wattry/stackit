# Code Style Rules

## General

- Clarity and simplicity is very important
- Unless specified we do not care about backwards compatability

## Go Code Style

- Follow standard Go conventions and idioms
- Prefer early returns over deep nesting
- Use meaningful variable names; avoid single-letter names except for loop indices

## Error Handling

- Always handle errors explicitly; never ignore them with `_`
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return errors to callers rather than logging and continuing

## Comments

- Write comments for exported functions and types
- Explain "why" not "what" in implementation comments
- Keep comments up to date when code changes
- Do not use low value comments, comments should only be required where the code is not self documenting

## Testing

- Use table-driven tests for multiple cases
- Keep test setup minimal and focused
