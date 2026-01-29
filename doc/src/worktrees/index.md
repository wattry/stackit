---
icon: material/folder-multiple
title: Git Worktrees with Stackit
description: Work on multiple stacks in parallel using Git worktrees. Create isolated directories for different features without stashing.
---

# Worktrees

Worktrees let you work on multiple stacks in parallel, each in its own directory. Instead of stashing changes or juggling branches, you can have separate working directories for different features.

## When to Use Worktrees

Worktrees are ideal when you need to:

- **Work on multiple features simultaneously** — Switch between projects without stashing
- **Review code while continuing development** — Keep your work-in-progress separate from the PR you're reviewing
- **Run long CI validation** — Test changes in an isolated directory while working elsewhere
- **Build on someone else's stack** — Checkout a teammate's PR without affecting your main workspace

For single-feature development, the standard single-directory workflow is simpler.

## Quick Start

```bash
# Create a worktree and open it
stackit worktree create my-feature --open

# You're now in ../your-repo-stacks/my-feature/
# Create branches as usual
git add feature.go
stackit create add-feature -m "feat: add new feature"
```

<div class="grid cards" markdown>

-   :material-play-circle:{ .lg .middle } **Getting Started**

    ---

    Create your first worktree, understand the basics, and learn different creation methods.

    [Getting started →](getting-started.md)

-   :material-console:{ .lg .middle } **Shell Integration**

    ---

    Enable automatic directory changes when opening or creating worktrees.

    [Shell integration →](shell-integration.md)

-   :material-cog:{ .lg .middle } **Management**

    ---

    List, open, remove worktrees. Configure base paths, auto-cleanup, and post-create hooks.

    [Management →](management.md)

</div>

## Key Commands

| Command | Description |
|---------|-------------|
| `stackit worktree create <name>` | Create a new worktree |
| `stackit worktree open <name>` | Open an existing worktree |
| `stackit worktree list` | List all worktrees |
| `stackit worktree remove <name>` | Remove a worktree |

## Related

- [Shell Integration (Integrations) →](../integrations/shell.md)
- [Daily Workflows →](../workflows/daily.md)
- [Configuration →](../cli/config.md)
