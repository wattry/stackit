---
icon: material/help-circle
title: Troubleshooting Stackit
description: Solutions to common stackit issues including merge conflicts, detached HEAD, failed restacks, permission errors, and recovery commands.
---

# Troubleshooting

Common issues and their solutions.

## "Branch not tracked by stackit"

If you see this error, the branch wasn't created with $$stackit create$$.

**Solution**: Manually track it:

```bash
stackit track
```

## Merge conflicts during restack

When $$stackit restack$$ encounters conflicts:

1. Resolve the conflicts in your editor
2. Stage the resolved files:
   ```bash
   git add .
   ```
3. Continue the restack:
   ```bash
   stackit continue
   ```

**Alternative**: Abort and try a different approach:

```bash
stackit abort
```

## Recovering from a failed operation

Stackit automatically saves state before operations.

**Solution**: Undo the last operation:

```bash
stackit undo
```

This restores branches and metadata to the state before the last command.

## Stack is out of sync with remote

If your local stack diverged from remote (e.g., after force-pushes by collaborators):

**Solution**: Sync with remote:

```bash
stackit sync
```

This pulls the latest trunk, cleans up merged branches, and restacks.

## PR base branch is wrong on GitHub

If a PR's base branch is pointing to the wrong parent:

**Solution**: Update all PRs in the stack:

```bash
stackit submit --stack
```

This updates all PRs with correct base branches.

## "Cannot modify frozen/locked branch"

Frozen or locked branches are protected from modification.

**Solution**: Unfreeze or unlock the branch:

```bash
# For locally frozen branches
stackit unfreeze <branch>

# For shared locked branches
stackit unlock <branch>
```

!!! warning
    Only unlock shared branches if you're authorized to modify them.

## Detached HEAD state

If you end up in a detached HEAD state:

**Solution**: Return to a tracked branch:

```bash
stackit checkout <branch-name>
```

Or return to trunk:

```bash
stackit trunk
```

## Missing GitHub CLI (`gh`)

Stackit requires the GitHub CLI for PR operations.

**Solution**: Install the GitHub CLI:

=== "macOS"

    ```bash
    brew install gh
    ```

=== "Linux"

    ```bash
    # See: https://github.com/cli/cli/blob/trunk/docs/install_linux.md
    ```

Then authenticate:

```bash
gh auth login
```

## Permission errors when pushing

If you get permission errors when running $$stackit submit$$:

**Solution**: Check your GitHub authentication:

```bash
gh auth status
```

If needed, re-authenticate:

```bash
gh auth login
```

## Worktree already exists

If you try to create a worktree that already exists:

**Solution**: Remove the existing worktree first:

```bash
stackit worktree remove <stack-name>
```

Or navigate to the existing worktree:

```bash
# With shell integration
stackit worktree open <stack-name>

# Without shell integration
cd $(stackit worktree open <stack-name>)
```

## Doctor command

For a comprehensive diagnosis of common issues:

```bash
stackit doctor
```

This checks for:

- Git version compatibility
- GitHub CLI installation and authentication
- Repository initialization
- Metadata integrity
- Common configuration issues

## Getting more help

If you're still stuck:

1. **Check the FAQ**: [community/faq.md](../community/faq.md)
2. **Enable debug mode**: Run commands with `--debug` for detailed output
3. **View debug information**: Run $$stackit debug$$ to see recent command history
4. **File an issue**: [github.com/getstackit/stackit/issues](https://github.com/getstackit/stackit/issues)

## Debug mode

For detailed diagnostic output, use the `--debug` flag:

```bash
stackit --debug restack
```

This shows:

- Git commands being executed
- Metadata operations
- Decision-making logic
- Error details

## Next steps

- [View all CLI commands →](../cli/reference.md)
- [Check the FAQ →](../community/faq.md)
- [Workflows →](../workflows/index.md)
