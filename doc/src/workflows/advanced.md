---
icon: material/tools
title: Advanced Stackit Workflows
description: Power-user operations including splitting commits, reorganizing stacks, moving branches, and running commands across multiple branches.
---

# Advanced Workflows

Power-user operations for reorganizing and managing complex stacks.

## Splitting Commits into Branches

Split commits from the current branch into separate branches using different modes:

### By Commit (default)

```bash
stackit split
```

Launches an interactive UI to select which commits to extract into a new branch.

### By Hunk

```bash
stackit split --by-hunk
```

Interactively select individual hunks (portions of files) to extract.

### By File

```bash
stackit split --by-file
```

Select entire files to extract into the new branch.

### Split Direction

By default, split creates a new branch **below** (as a parent). Use `--above` to create the branch **above** (as a child):

```bash
stackit split --above -m "extract feature"
```

### Preview Changes

Use `--dry-run` to see what would happen without making changes:

```bash
stackit split --dry-run
```

## Reorganizing a Stack

To change the order of branches in your stack:

```bash
stackit reorder
```

This shows an interactive editor where you can reorder branches. Stackit handles rebasing automatically.

## Moving a Branch to a New Parent

To move a branch (and its children) onto a different parent:

```bash
stackit move <source-branch> <new-parent>
```

Example: Move `feature-ui` from building on `feature-api` to building on `main`:

```bash
stackit move feature-ui main
```

## Extracting a Branch from a Stack

To remove a branch from the middle of a stack without affecting its children:

```bash
stackit pluck <branch>
```

This reparents the children to the branch's grandparent, effectively removing it from the stack.

## Running Commands Across the Stack

Execute a shell command on each branch in the stack:

```bash
stackit foreach <command>
```

Example: Run tests on all branches:

```bash
stackit foreach "go test ./..."
```

By default, this runs on all upstack branches (from current to top). Use `--downstack` to run from current to trunk.

### CI Validation

Configure a CI command in `.stackit.yaml` for consistent validation:

```yaml
# .stackit.yaml
ci:
  command: "make test"
  timeout: 600  # seconds
```

Then run it across your stack:

```bash
stackit foreach
```

Or run any command:

```bash
# Run on all upstack branches
stackit foreach "npm test"

# Run on downstack branches instead
stackit foreach --downstack "go test ./..."
```

## Worktree Automation

Configure commands to run automatically after worktree creation:

```yaml
# .stackit.yaml
hooks:
  post-worktree-create:
    - npm install
    - cp .env.example .env
```

See [Worktrees Management](../worktrees/management.md) for more details.

## Creating Worktrees from Within Worktrees

You can create new worktrees even when inside a managed worktree:

```bash
# Inside an existing worktree (e.g., ../your-repo-stacks/feature-a/)
stackit create another-feature -m "feat: another feature" -w
```

This creates a sibling worktree at `../your-repo-stacks/another-feature/` regardless of where you're currently working. The new branch is created from trunk, not from the current worktree's branch.

## Next Steps

- [Daily Workflows →](daily.md)
- [Team Collaboration →](collaboration.md)
- [CLI Reference →](../cli/reference.md)
