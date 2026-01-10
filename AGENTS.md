# Stackit

Go-based CLI tool for managing stacked changes in Git repositories.

**Tech stack:** Go 1.25, Cobra, Just (justfile)

## Architecture

- `cmd/stackit`: CLI entry point and command definitions.
- `internal/actions`: High-level business logic for CLI commands (create, submit, sync, undo, etc.).
- `internal/engine`: Core logic for managing stacked branches, metadata state, and branch relationships.
- `internal/git`: Low-level Git operations (executing git commands, reading config, managing refs).
- `internal/utils`: Shared utilities for branch naming, sanitization, and UI helpers.

## Requirements

**All changes must pass tests and lint before committing:**

```bash
just check             # Runs fmt, lint, and fast tests (recommended)
just test-fast         # Run fast unit tests (~30s)
just test-integration  # Run integration tests (~90s)
just test              # Run all tests
just test-pkg ./pkg    # Run tests for a specific package (e.g. ./internal/git)
just lint              # Run linter
```

**Workflow:** Run `just check` during development for quick feedback. Run `just test` before submitting PRs to ensure all tests pass.

## Build

```bash
just build   # Builds ./stackit binary
just deps    # Install dependencies
```

## Commit Messages

Use **Conventional Commits**:

```
<type>[optional scope]: <description>
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`

**Examples:**
- `feat: add branch traversal functionality`
- `fix: resolve merge conflict detection issue`
- `refactor: simplify merge plan logic`

## Using Stackit for Commits and PRs

This repo uses stackit for managing stacked changes. **NEVER use raw git commands for branch/commit operations. Always use `command stackit`.**

### Critical Rules

1. **NEVER use `git commit` directly** - Always use `command stackit create` for new commits
2. **NEVER use `git checkout -b`** - Use `command stackit create` to create branches
3. **NEVER use `gh pr create`** - Use `command stackit submit` for PRs

| Forbidden | Required |
|-----------|----------|
| `git commit -m "..."` | `command stackit create -m "..."` |
| `git checkout -b branch` | `command stackit create -m "..."` |
| `gh pr create` | `command stackit submit` |
| `git rebase` | `command stackit restack` |

### Correct Workflow for Creating Stacked Changes

**IMPORTANT:** `command stackit create` requires staged changes. You MUST stage changes BEFORE running the command.

```bash
# 1. Check current stack state
command stackit log

# 2. Make your code changes
# ... edit files ...

# 3. Stage your changes FIRST
git add -A

# 4. Create the stacked branch WITH the commit (staged changes required!)
command stackit create -m "feat: my feature description"

# 5. Submit PRs when ready
command stackit submit
```

### Common Mistakes to Avoid

```bash
# WRONG - Creates empty branch, then commits outside stackit
command stackit create -m "feat: something"  # No staged changes = empty branch!
# ... make changes ...
git commit -m "feat: something"      # WRONG! Bypasses stackit

# CORRECT - Stage first, then create
# ... make changes ...
git add -A
command stackit create -m "feat: something"  # Creates branch + commits staged changes
```

### Available Skills

Run `/stackit` for the full guide, or use specific skills:
- `/stack-create` - Create a stacked branch (use this instead of manual commands)
- `/stack-submit` - Submit PRs
- `/stack-status` - View stack health
- `/stack-fix` - Diagnose and fix issues
- `/stack-sync` - Sync with trunk and cleanup

## Implementation Details

### Metadata Handling
Stackit manages branch relationships and PR state using custom Git references and notes. 
- **Branch Metadata**: Stored in `refs/stackit/metadata/` for each branch.
- **PR Information**: Managed through the `Engine` which abstracts the storage of PR titles, bodies, and status.
- **State Management**: The `internal/engine` package is the source of truth for the stack structure. Always use the `Engine` to query or modify branch relationships.

