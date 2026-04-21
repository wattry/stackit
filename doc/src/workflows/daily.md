---
icon: material/clock-outline
title: Daily Stackit Workflows
description: Common daily tasks with stackit including updating after code review, using absorb for multi-branch fixes, and syncing with main.
---

# Daily Workflows

Common tasks you'll perform regularly when working with stacked changes.

## Updating After Code Review

When you receive feedback on a branch in the middle of your stack:

1. Navigate to that branch:
   ```bash
   stackit checkout <branch>
   ```

2. Make your changes and amend:
   ```bash
   stackit modify
   ```

3. Update child branches:
   ```bash
   stackit restack --upstack
   ```

4. Update the PRs:
   ```bash
   stackit submit
   ```

## Using Absorb for Multi-Branch Fixes

$$stackit absorb$$ is like magic for stacked PRs. If you have small fixes for multiple branches in your stack, just stage them all and run absorb.

Example scenario: You notice a typo in `add-api` and a bug in `add-logic`:

```bash
# Make fixes to multiple files
git add internal/api.go internal/logic.go

# Intelligently amend to the correct branches
stackit absorb
```

Stackit figures out which changes belong to which branch and amends them automatically.

## Syncing with the Main Branch

To keep your stack up-to-date with `main`:

```bash
stackit sync
```

This command:

1. Pulls the latest changes from `main`
2. Deletes branches that have already been merged
3. Restacks your remaining branches on top of the new `main`

!!! tip
    Run $$stackit sync$$ regularly to stay current with trunk.

## Flattening a Stack

After landing PRs from the middle of a stack, use $$stackit flatten$$ to move branches closer to trunk:

```bash
stackit flatten
```

This analyzes each branch and tests whether it can be rebased directly onto trunk (or closer to it). Branches that depend on changes from their parent will stay in place.

### Before Flattening

```
● feature-c
│
◯ feature-b (merged)
│
◯ feature-a (merged)
│
main
```

### After Flattening

```
● feature-c
│
main
```

## Working on Multiple Stacks

To work on separate features simultaneously, each in their own directory:

```bash
# Create a new stack with its own worktree
stackit create my-feature -m "feat: start new feature" -w
```

This creates:

- A new branch `my-feature` tracked by stackit
- A worktree at `../your-repo-stacks/my-feature/`

Navigate to the worktree:

```bash
# With shell integration: auto-changes directory
stackit worktree open my-feature

# Without shell integration: use command substitution
cd $(stackit worktree open my-feature)
```

See the [Worktrees Guide](../worktrees/index.md) for comprehensive documentation.

## Next Steps

- [Advanced Workflows →](advanced.md)
- [Team Collaboration →](collaboration.md)
- [Worktrees →](../worktrees/index.md)
