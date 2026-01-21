# Documentation Rules

## When to Update Docs

- New CLI commands → Add to README.md Command Reference
- New workflows → Add examples to Common Workflows in README.md
- Configuration changes → Update `docs/config.md`
- TUI changes → Update `docs/tui.md`

## Command Help Text

The `Long` description in Cobra commands should include concrete examples:

```go
Long: `Syncs your stack with the remote repository.

Examples:
  stackit sync              # Sync current stack
  stackit sync --all        # Sync all branches`,
```

## Technical Docs (`docs/`)

- `docs/config.md` - Configuration keys, layered config, adding new keys
- `docs/tui.md` - TUI patterns, styling, components

Keep these up-to-date when modifying related systems.
