---
icon: material/download
---

# Installation

## Homebrew (macOS and Linux)

The easiest way to install stackit is via Homebrew:

```bash
brew install getstackit/tap/stackit
```

After installation, you can use either `stackit` or `st` (short alias).

## Shell integration

!!! tip "Recommended"
    Enable shell integration to automatically change directories when creating worktrees with `stackit create -w`.

Add one of the following to your shell configuration:

=== "zsh"

    ```bash
    # ~/.zshrc
    eval "$(stackit shell zsh)"
    ```

=== "bash"

    ```bash
    # ~/.bashrc
    eval "$(stackit shell bash)"
    ```

=== "fish"

    ```fish
    # ~/.config/fish/config.fish
    stackit shell fish | source
    ```

This is separate from shell completions. You likely want both:

```bash
# zsh example:
eval "$(stackit completion zsh)"
eval "$(stackit shell zsh)"
```

## Shell completions

Enable tab completion for stackit commands:

=== "zsh"

    ```bash
    # ~/.zshrc
    eval "$(stackit completion zsh)"
    ```

=== "bash"

    ```bash
    # ~/.bashrc
    eval "$(stackit completion bash)"
    ```

=== "fish"

    ```fish
    # ~/.config/fish/config.fish
    stackit completion fish | source
    ```

## Verify installation

Check that stackit is installed correctly:

```bash
stackit --version
```

## Next steps

- [Create your first stack →](stack.md)
- [Learn core concepts →](../guide/concepts.md)
