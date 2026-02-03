---
icon: material/cog
title: Worktree Management
description: List, open, remove, and configure stackit worktrees. Set up auto-cleanup, custom paths, and post-create hooks for dependency installation.
---

# Worktree Management

List, configure, and maintain your worktrees.

## Listing Worktrees

View all stackit-managed worktrees:

```bash
stackit worktree list
```

Shows each worktree's anchor branch, path, and status.

## Opening an Existing Worktree

Switch to a worktree directory:

```bash
stackit worktree open my-feature
```

You can specify either the worktree name or the anchor branch name.

!!! note
    Requires [shell integration](shell-integration.md) for automatic directory change. Without it, use:
    ```bash
    cd $(stackit worktree open my-feature)
    ```

## Removing a Worktree

Clean up a worktree when you're done:

```bash
stackit worktree remove my-feature
```

This removes:

- The worktree directory
- The worktree registration in stackit

The stack's branches remain intact. Use `--force` to remove even if there are errors.
Use `--keep-branch` to preserve the anchor branch when removing the worktree.

## Attaching to an Existing Stack

Attach an existing branch/stack to a new worktree:

```bash
stackit worktree attach my-feature
```

This creates a worktree for an existing stack that wasn't originally created with `-w`. Useful when you want to work on an existing stack in isolation.

## Detaching a Worktree

Detach a worktree without removing it from disk:

```bash
stackit worktree detach my-feature
```

This unregisters the worktree from stackit tracking while leaving the directory intact. Use `--force` if there are uncommitted changes.

## Pruning Stale Worktrees

Clean up worktrees that no longer exist on disk:

```bash
stackit worktree prune
```

Use `--dry-run` to preview what would be cleaned up without making changes.

## Automatic Cleanup

During $$stackit sync$$, worktrees for merged stacks are automatically cleaned up when `worktree.autoClean` is enabled (the default).

## Configuration Options

### worktree.basePath

Customize where worktrees are created:

```bash
stackit config set worktree.basePath "../my-stacks"
```

**Default**: `../<repo-name>-stacks`

### worktree.autoClean

Control automatic worktree cleanup during sync:

```bash
stackit config set worktree.autoClean false
```

**Default**: `true`

## Post-Create Hooks

Run commands automatically after worktree creation by adding a `.stackit.yaml` file:

```yaml
# .stackit.yaml
hooks:
  post-worktree-create:
    - npm install
    - cp .env.example .env
```

Common uses:

- Installing dependencies
- Setting up environment files
- Running initialization scripts

!!! warning "Security"
    The first time a hook is encountered, stackit prompts for approval. Approvals are stored locally and persist across sessions.

See [Configuration](../cli/config.md#worktree-hooks) for more examples.

## Working in Worktrees

### Creating Stacked Branches

Once inside a worktree, create branches as usual:

```bash
# Make changes
git add feature.go

# Create a stacked branch
stackit create add-feature -m "feat: add new feature"
```

### Creating Worktrees from Worktrees

You can create new worktrees from inside an existing worktree:

```bash
# Inside ../repo-stacks/feature-a/
stackit worktree create feature-b --open
```

Stackit detects the context and creates the new worktree from the main repository. The new worktree is a sibling, not nested.

### Returning to Main Repository

```bash
# From inside a worktree
cd $(git rev-parse --path-format=absolute --git-common-dir)/..

# Or simply navigate to your original repo path
cd ~/projects/my-repo
```

## Best Practices

- **One worktree per stack** — Keep features isolated for easier context switching
- **Clean up after merging** — Remove worktrees once PRs are merged, or let $$stackit sync$$ do it automatically
- **Use scopes** — Associate worktrees with tickets using `--scope` for better organization
- **Set up hooks** — Configure `post-worktree-create` hooks to automate dependency installation

## Next Steps

- [Getting Started →](getting-started.md)
- [Shell Integration →](shell-integration.md)
- [Configuration →](../cli/config.md)
