---
icon: material/tools
---

# Advanced Workflows

Power-user operations for reorganizing and managing complex stacks.

## Splitting Commits into Branches

If you have multiple commits on a branch and want to split them into separate branches:

```bash
stackit split
```

This launches an interactive UI where you can reorganize commits into multiple branches.

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
