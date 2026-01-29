---
icon: material/console
title: Shell Integration for Worktrees
description: Enable automatic directory changes when opening or creating stackit worktrees. Quick setup for zsh, bash, and fish shells.
---

# Shell Integration for Worktrees

Shell integration enables automatic directory changes when opening or creating worktrees. Without it, you need to manually `cd` into worktree directories.

## Quick Setup

Add the appropriate line to your shell config:

=== "zsh"

    ```bash
    # Add to ~/.zshrc
    eval "$(stackit shell zsh)"
    ```

=== "bash"

    ```bash
    # Add to ~/.bashrc
    eval "$(stackit shell bash)"
    ```

=== "fish"

    ```fish
    # Add to ~/.config/fish/config.fish
    stackit shell fish | source
    ```

Then restart your shell or source the config.

## What It Enables

With shell integration, worktree commands change your working directory automatically:

```bash
stackit worktree create my-feature --open  # Creates and cd's into worktree
stackit worktree open my-feature           # Opens existing worktree
```

## Without Shell Integration

Navigate manually using command substitution:

```bash
cd $(stackit worktree open my-feature)
```

## Full Guide

For complete setup instructions, troubleshooting, and how it works:

**[Shell Integration (Full Guide) →](../integrations/shell.md)**

## Next Steps

- [Getting Started with Worktrees →](getting-started.md)
- [Worktree Management →](management.md)
