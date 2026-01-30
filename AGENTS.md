# Stackit

Go CLI for managing stacked changes in Git repositories.

**Tech stack:** Go 1.25, Cobra, bubbletea, mise

## Architecture

```
cmd/stackit/       CLI entry point
internal/
├── actions/       Business logic for commands (create, submit, sync, etc.)
├── cli/           Command definitions, handlers
├── engine/        Core stack logic, metadata, branch relationships
├── git/           Low-level git operations
├── tui/           Terminal UI components (see docs/tui.md)
├── config/        Configuration system (see docs/config.md)
└── utils/         Shared utilities
```

## Documentation

- `docs/config.md` - Configuration system, keys, layered config
- `docs/tui.md` - TUI patterns, BaseModel, styling, components
- `docs/worktree.md` - Worktree management, create vs attach, workflows

## Common Development Tasks

| Task | Guide |
|------|-------|
| Add a new config option | See "Adding New Configuration" in `docs/config.md` |
| Add submit command config | See "Submit Command Config Flow" in `docs/config.md` |
| Add a new CLI command | Follow patterns in `internal/cli/` |
| Add TUI component | See `docs/tui.md` |
| Add worktree functionality | See `docs/worktree.md` |

## CLI Tools

Use these tools instead of standard alternatives for better performance:

| Tool | Use Instead Of | Purpose |
|------|----------------|---------|
| `rg` (ripgrep) | `grep` | Fast text search, respects .gitignore |
| `fd` | `find` | Fast file search, intuitive syntax |
| `ast-grep` | `sed` for code | AST-based code search and refactoring |
| `jq` | - | JSON processing |
| `yq` | - | YAML processing |
| `tokei` | `wc -l` | Code statistics and language breakdown |

**Examples:**
```bash
rg "func.*Create" --type go          # Search for Create functions in Go files
fd "\.go$" internal/                  # Find all Go files in internal/
ast-grep -p 'fmt.Errorf($$$)' .       # Find all fmt.Errorf calls
tokei                                 # Get codebase statistics
jq '.dependencies' package.json       # Parse JSON
```

## Requirements

**All changes must pass tests and lint before committing:**

```bash
mise run check         # Runs fmt, lint, and fast tests
mise run test-fast     # Run fast unit tests (~30s)
mise run test-integration  # Run integration tests (~90s)
mise run test          # Run all tests
mise run test-pkg ./pkg    # Run tests for a specific package (e.g. ./internal/git)
mise run lint          # Run linter
```

**Workflow:** Run `mise run check` during development for quick feedback. Run `mise run test` before submitting PRs to ensure all tests pass.

## Build

```bash
mise run build   # Builds ./stackit binary
mise run deps    # Install dependencies
```

## Commit Messages

Use **Conventional Commits**: `<type>: <description>`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`

## Stacking Multi-Phase Changes

**When implementing changes with multiple logical phases, use stackit to create separate PRs for each phase.** This makes code review easier and keeps each PR focused.

**When to stack:**
- Adding multiple related features (e.g., new phase + error handling + docs)
- Changes that touch different subsystems
- Refactoring + new feature in the same task
- Any change with 3+ distinct logical units

**Example:** Adding dependency analysis to a skill file:
```bash
# Phase 1: Add dependency analysis
git add -A && command stackit create -m "feat: add dependency analysis phase"

# Phase 2: Enhance related file detection
git add -A && command stackit create -m "feat: add related file detection guidance"

# Phase 3: Update error handling
git add -A && command stackit create -m "feat: add error cases for new phases"

# Submit all as stacked PRs
command stackit submit
```

**Benefits:**
- Reviewers can approve phases independently
- Easy to revert a single phase if needed
- Clearer git history
- Smaller, focused diffs

**Use `/stack-plan` when starting from scratch** to plan the stack structure before writing code.

## Metadata

Stackit stores branch relationships in `refs/stackit/metadata/`. The `Engine` package is the source of truth for stack structure. Always use `Engine` to query/modify branch relationships.

