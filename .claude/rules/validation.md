# Validation Strategy

Use the lightest validation that covers your change. Running full test suites for every edit wastes time and context.

## Validation Levels (Fastest to Slowest)

| Level | Command | Time | Use When |
|-------|---------|------|----------|
| Compile | `mise run compile` | ~2s | Docs, comments, type changes |
| Lint | `mise run lint` | ~5s | Refactoring, style changes |
| Package tests | `mise run test:pkg ./internal/foo` | ~10s | Single package logic changes |
| Fast tests | `mise run check` | ~30s | Multi-package logic changes |
| Full suite | `mise run test` | ~2min | Engine/integration changes |
| Web tests | `mise run web:test` | ~10s | Web component changes |
| Web full | `mise run check:web` | ~30s | Web + API contract changes |

## Decision Guide

### Use Compile Check (`mise run compile`)

- Changed comments or documentation
- Fixed typos in strings
- Added/removed imports
- Changed type definitions without behavior changes
- Quick "does it build?" verification

### Use Lint Only (`mise run lint`)

- Renamed variables or functions
- Extracted helper functions
- Reorganized code within a file
- Style/formatting changes
- Changes that don't affect behavior

### Use Package Tests (`mise run test:pkg ./internal/foo`)

- Changed logic in a single package
- Fixed a bug in one module
- Added a new function to an existing package
- Modified internal implementation details

### Use Fast Tests (`mise run check`)

- Changes spanning multiple packages
- Modified public interfaces
- Added new CLI commands
- Changed configuration handling

### Use Full Suite (`mise run test`)

- Engine changes (branch relationships, metadata)
- Git operation changes
- Integration test changes
- Before submitting PRs

## Examples

```bash
# Fixed a typo in a comment
mise run compile

# Renamed a variable in git package
mise run lint

# Added new helper function to engine
mise run test:pkg ./internal/engine

# Changed how submit processes branches
mise run check

# Modified branch relationship logic
mise run test

# Changed a React component in apps/web
mise run web:test

# Changed API response types used by frontend
mise run check:web
```

### Use Web Tests (`mise run web:test`)

- Changed a React component
- Modified hooks or lib utilities
- Updated styles that affect component behavior

### Use Web Full (`mise run check:web`)

- Changed API client or response types
- Modified data flow (providers, SSE)
- Changes spanning web + Go API contracts

## Escalation

If a lighter validation passes but you're uncertain, escalate to the next level. Trust your judgment about what the change might affect.

**Rule of thumb:** Match validation scope to change scope.
