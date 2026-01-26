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

### Initialize team configuration

Create a `.stackit.yaml` file with all available options:

```bash
stackit config init
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

!!! note "Auto-generated content"
    The options below are auto-generated from the config metadata registry.

--8<-- "config-reference.md"

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
- [Shell Integration →](shell.md)
- [Learn common workflows →](../guide/workflows.md)
- [Worktrees Guide →](../guide/worktrees.md)
- [Team Workflows →](../guide/team-workflows.md)
