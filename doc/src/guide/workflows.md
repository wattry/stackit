---
icon: material/cog
---

# Common Workflows

Real-world examples of using stackit effectively.

## Updating after code review

If you receive feedback on a branch in the middle of your stack:

1. Navigate to that branch:
   ```bash
   stackit checkout <branch>
   ```

2. Make your changes and amend:
   ```bash
   stackit modify
   ```

3. Update all child branches:
   ```bash
   stackit restack
   ```

4. Update the PRs:
   ```bash
   stackit submit
   ```

## Using absorb for multi-branch fixes

$$stackit absorb$$ is like magic for stacked PRs. If you have small fixes for multiple branches in your stack, just stage them all and run absorb.

Example scenario: You notice a typo in `add-api` and a bug in `add-logic`:

```bash
# Make fixes to multiple files
git add internal/api.go internal/logic.go

# Intelligently amend to the correct branches
stackit absorb
```

Stackit figures out which changes belong to which branch and amends them automatically.

## Flattening a stack

After landing PRs from the middle of a stack, use $$stackit flatten$$ to move branches closer to trunk:

```bash
stackit flatten
```

This analyzes each branch and tests whether it can be rebased directly onto trunk (or closer to it). Branches that depend on changes from their parent will stay in place.

### Before flattening

```
● feature-c
│
◯ feature-b (merged)
│
◯ feature-a (merged)
│
main
```

### After flattening

```
● feature-c
│
main
```

## Syncing with the main branch

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

## Working on multiple stacks in parallel

To work on separate features simultaneously, each in their own directory:

```bash
# Create a new stack with its own worktree
stackit create my-feature -m "feat: start new feature" -w
```

This creates:

- A new branch `my-feature` tracked by stackit
- A worktree at `../your-repo-stacks/my-feature/`

Navigate to the worktree with:

```bash
cd $(stackit worktree open my-feature)
```

### Automating worktree setup

Configure commands to run automatically after worktree creation by adding a `.stackit.yaml` file to your repository:

```yaml
# .stackit.yaml
hooks:
  post-worktree-create:
    - npm install
    - cp .env.example .env
```

These hooks run in the new worktree directory after it's created. See [Configuration](../cli/config.md#project-configuration-stackityaml) for more details.

Worktrees are automatically cleaned up during $$stackit sync$$ when their stack is merged.

## Collaborating on stacks

### Fetching someone's stack

To work on a stack created by someone else:

```bash
# Sync an entire stack by providing a PR number or branch name
stackit get 123
```

By default, $$stackit get$$ **freezes** the fetched branches locally. This prevents accidental local modifications while you build on top of them, without affecting the original author's metadata.

### Building on top of a fetched stack

After fetching, you can create your own branches on top:

```bash
stackit checkout <top-branch-of-their-stack>
git add your-changes.go
stackit create your-feature -m "feat: your addition"
```

### Unfreezing to make changes

If you need to modify the fetched branches:

```bash
stackit unfreeze <branch>
```

## Splitting commits into branches

If you have multiple commits on a branch and want to split them into separate branches:

```bash
stackit split
```

This launches an interactive UI where you can reorganize commits into multiple branches.

## Reorganizing a stack

To change the order of branches in your stack:

```bash
stackit reorder
```

This shows an interactive editor where you can reorder branches. Stackit handles rebasing automatically.

## Moving a branch to a new parent

To move a branch (and its children) onto a different parent:

```bash
stackit move <source-branch> <new-parent>
```

Example: Move `feature-ui` from building on `feature-api` to building on `main`:

```bash
stackit move feature-ui main
```

## Extracting a branch from a stack

To remove a branch from the middle of a stack without affecting its children:

```bash
stackit pluck <branch>
```

This reparents the children to the branch's grandparent, effectively removing it from the stack.

## Running commands across the stack

Execute a shell command on each branch in the stack:

```bash
stackit foreach <command>
```

Example: Run tests on all branches:

```bash
stackit foreach "go test ./..."
```

By default, this runs on all upstack branches (from current to top). Use `--downstack` to run from current to trunk.

## Next steps

- [Setup Claude integration →](claude-integration.md)
- [Troubleshooting →](troubleshooting.md)
