---
icon: material/wrench
---

# Configuration

Customize stackit behavior with configuration options.

## Managing configuration

### Interactive configuration

Use the interactive TUI to manage all settings:

```bash
stackit config
```

### List current configuration

View all current configuration values:

```bash
stackit config --list
```

### Set a value

```bash
stackit config set <key> <value>
```

### Get a value

```bash
stackit config get <key>
```

### Remove a value

```bash
stackit config unset <key>
```

## Available options

### branch.pattern

Customize how branch names are generated when not explicitly specified.

**Default**: `{username}/{date}/{message}`

**Available placeholders**:
- `{username}` - Git username
- `{date}` - Current date (YYYYMMDD format)
- `{message}` - Sanitized commit message
- `{scope}` - Current scope (if set)

**Example**:

```bash
# Use Jira-style naming
stackit config set branch.pattern "{scope}/{message}"

# Use simple naming
stackit config set branch.pattern "{message}"
```

### submit.footer

Control whether PRs include a footer linking back to the stack.

**Default**: `true`

**Example**:

```bash
stackit config set submit.footer false
```

### worktree.basePath

Customize where worktrees are created.

**Default**: `../<repo-name>-stacks`

**Example**:

```bash
stackit config set worktree.basePath "../my-stacks"
```

### worktree.autoClean

Auto-remove worktrees for merged stacks during sync.

**Default**: `true`

**Example**:

```bash
stackit config set worktree.autoClean false
```

## Configuration file location

Stackit configuration is stored in `.git/config` under the `[stackit]` section.

You can also edit it directly:

```bash
git config --local stackit.submit.footer true
```

## Scope-specific configuration

Configuration is repository-specific by default. To set global defaults:

```bash
git config --global stackit.submit.footer false
```

## Next steps

- [View command reference →](reference.md)
- [Learn common workflows →](../guide/workflows.md)
