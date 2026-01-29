---
icon: material/console
title: Shell Integration
description: Enable stackit to change your working directory when opening worktrees. Setup guide for zsh, bash, and fish with troubleshooting tips.
---

# Shell Integration

Shell integration enables stackit to change your working directory when opening or creating worktrees. This is separate from tab completions.

## Setup

Add the appropriate line to your shell configuration file:

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

After adding the line, restart your shell or source the configuration:

```bash
source ~/.zshrc  # or ~/.bashrc
```

## Complete Setup

You likely want both shell integration and tab completions:

=== "zsh"

    ```bash
    # Add both to ~/.zshrc
    eval "$(stackit completion zsh)"
    eval "$(stackit shell zsh)"
    ```

=== "bash"

    ```bash
    # Add both to ~/.bashrc
    eval "$(stackit completion bash)"
    eval "$(stackit shell bash)"
    ```

=== "fish"

    ```fish
    # Add both to ~/.config/fish/config.fish
    stackit completion fish | source
    stackit shell fish | source
    ```

## Features Enabled

### Automatic Directory Changes

With shell integration, these commands change your working directory:

```bash
# Create and open a new worktree
stackit worktree create my-feature --open

# Open an existing worktree
stackit worktree open my-feature
```

### Interactive Prompts

In interactive mode, stackit prompts whether to open the worktree after creation:

```
Created worktree at ../repo-stacks/my-feature
Open worktree? [Y/n]
```

## Without Shell Integration

If you prefer not to use shell integration, you can still navigate to worktrees manually:

```bash
# Print path and navigate manually
cd $(stackit worktree open my-feature)

# Or use command substitution in the create flow
cd $(stackit worktree create my-feature)
```

## Troubleshooting

### Shell Integration Not Working

1. **Verify setup**: Check that the eval line is in your shell config

    ```bash
    grep "stackit shell" ~/.zshrc  # or ~/.bashrc
    ```

2. **Source the config**: Restart your shell or run:

    ```bash
    source ~/.zshrc
    ```

3. **Check for errors**: Run the shell command directly to see any output:

    ```bash
    stackit shell zsh
    ```

### Multiple Shell Configurations

If you use multiple shells, add integration to each:

- `~/.zshrc` for zsh
- `~/.bashrc` for bash
- `~/.config/fish/config.fish` for fish

### Directory Changes Not Persisting

Shell integration uses a shell function wrapper. If changes aren't persisting, ensure:

1. You're using `stackit` directly, not through a script or alias that spawns a subshell
2. The shell function is loaded (check with `type stackit`)

## How It Works

The shell integration defines a function that wraps the `stackit` command. When worktree commands output a special marker followed by a path, the wrapper function captures this and runs `cd` in your current shell.

This approach is necessary because child processes cannot change the parent shell's working directory directly.

## Next Steps

- [Worktrees →](../worktrees/index.md)
- [Worktree Shell Integration →](../worktrees/shell-integration.md)
- [Command Reference →](../cli/reference.md)
- [Other integrations →](index.md)
