---
icon: material/wrench
---

# Configuration

Customize stackit behavior with configuration options.

## Layered configuration

Stackit uses a layered configuration system:

1. **Personal settings** (highest priority) — Stored in `.git/config`, not shared
2. **Team settings** — Stored in `.stackit.yaml`, committed and shared with team
3. **Defaults** (lowest priority) — Built-in sensible defaults

This allows teams to define shared settings that individual developers can override locally.

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

### merge.method

Default merge strategy for PRs.

**Default**: `""` (GitHub default)

**Options**: `squash`, `merge`, `rebase`

**Example**:

```bash
stackit config set merge.method squash
```

### ci.command

CI validation command to run with `stackit foreach`.

**Default**: `""` (none)

**Example**:

```bash
stackit config set ci.command "make test"
```

### ci.timeout

CI command timeout in seconds.

**Default**: `600` (10 minutes)

**Example**:

```bash
stackit config set ci.timeout 300
```

## Team configuration (`.stackit.yaml`)

For team-wide settings that should be shared across all contributors, create a `.stackit.yaml` file in your repository root and commit it to version control. Team settings act as defaults that individual developers can override in their personal git config.

```yaml
# .stackit.yaml - Team-wide defaults
trunk: main

# Additional trunk branches (e.g., release branches)
trunks:
  - develop
  - staging

# Branch naming pattern for the team
branch:
  pattern: "{username}/{date}/{message}"

# PR submission settings
submit:
  footer: true

# Default merge method
merge:
  method: squash

# CI validation
ci:
  command: "make test"
  timeout: 600

# Worktree hooks
hooks:
  post-worktree-create:
    - npm install
    - cp .env.example .env
```

### Worktree hooks

The `hooks.post-worktree-create` option allows you to run commands automatically after creating a worktree with `stackit create -w`.

This is useful for:

- Installing dependencies (`npm install`, `bundle install`, `pip install -r requirements.txt`)
- Setting up environment files
- Running initialization scripts
- Configuring IDE settings

### Security

For safety, the first time a hook is encountered, stackit prompts for approval:

```
This repo wants to run "npm install" after creating worktrees. Allow? [y/N]
```

- The default answer is "No" for security
- Approvals are stored locally in git config
- Approvals persist across sessions
- Hooks have a 60-second timeout

### Example configurations

**Node.js project**:

```yaml
hooks:
  post-worktree-create:
    - npm install
```

**Python project**:

```yaml
hooks:
  post-worktree-create:
    - python -m venv .venv
    - .venv/bin/pip install -r requirements.txt
```

**Multi-step setup**:

```yaml
hooks:
  post-worktree-create:
    - npm install
    - cp .env.example .env
    - npm run setup
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
