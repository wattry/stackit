---
icon: material/folder-multiple
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

## Creating a Worktree

Create a new worktree from trunk:

```bash
stackit worktree create my-feature
```

This creates:

- A new anchor branch `my-feature` tracked by stackit
- A worktree directory at `../<repo-name>-stacks/my-feature/`

### Opening After Creation

Use `--open` to automatically change to the new worktree directory:

```bash
stackit worktree create my-feature --open
```

!!! note "Shell Integration Required"
    The `--open` flag requires [shell integration](#shell-integration) to change your working directory. Without it, stackit prints the path instead.

### Setting a Scope

Associate a Jira ticket or Linear issue with all branches in the worktree:

```bash
stackit worktree create my-feature --scope PROJ-123
```

All branches created in this worktree will include the scope in their names.

### Alternative: Create During Branch Creation

You can also create a worktree while creating your first branch:

```bash
git add .
stackit create my-feature -m "feat: start new feature" -w
```

## Shell Integration

Shell integration enables automatic directory changes when opening or creating worktrees.

### Setup

Add one of these to your shell configuration:

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

!!! tip "Don't Forget Completions"
    Shell integration is separate from tab completions. You probably want both:

    ```bash
    eval "$(stackit completion zsh)"
    eval "$(stackit shell zsh)"
    ```

### What It Enables

With shell integration:

- `stackit worktree create --open` changes to the new directory
- `stackit worktree open` changes to the existing directory
- In interactive mode, you're prompted to open worktrees after creation

Without shell integration, these commands print the path for manual navigation:

```bash
cd $(stackit worktree open my-feature)
```

See [Shell Integration](../integrations/shell.md) for more details.

## Managing Worktrees

### List All Worktrees

View all stackit-managed worktrees:

```bash
stackit worktree list
```

Shows each worktree's anchor branch, path, and status.

### Open an Existing Worktree

Switch to a worktree directory:

```bash
stackit worktree open my-feature
```

You can specify either the worktree name or the anchor branch name.

### Remove a Worktree

Clean up a worktree when you're done:

```bash
stackit worktree remove my-feature
```

This removes:

- The worktree directory
- The worktree registration in stackit

The stack's branches remain intact. Use `--force` to remove even if there are errors.

### Automatic Cleanup

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

- [Shell Integration →](../integrations/shell.md)
- [Common Workflows →](workflows.md)
- [Configuration →](../cli/config.md)
