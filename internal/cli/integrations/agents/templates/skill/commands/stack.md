# Stack Operations Commands

Commands for managing the entire stack or multiple branches.

> **CRITICAL:** Always run these commands with `command stackit ... --no-interactive`.

## Stack Maintenance

| Command | Description |
|---------|-------------|
| `command stackit restack --branch <branch> --upstack --no-interactive` | Rebase a branch and its descendants (preferred — minimizes churn) |
| `command stackit restack --all-stacks --continue-on-conflict --no-interactive` | Rebase every independent stack rooted at trunk, skipping conflicted stacks so unrelated stacks still proceed |
| `command stackit restack --stacks <root1>,<root2> --continue-on-conflict --no-interactive` | Rebase specific independent stack roots while letting unrelated selected roots continue past conflicts |
| `command stackit sync --no-interactive` | Pull trunk, delete merged branches, restack |
| `command stackit info --stack --json --no-interactive` | Export full stack metadata as JSON for analysis |
| `command stackit merge` | Merge approved PRs and cleanup |
| `command stackit fold --no-interactive` | Fold current branch into its parent |

## Bulk Operations

| Command | Description |
|---------|-------------|
| `command stackit foreach --no-interactive` | Run command on each branch in stack |
| `command stackit submit --no-interactive` | Push branches and create/update PRs |
| `command stackit reorder` | Interactively reorder branches |
| `command stackit move` | Rebase branch onto new parent |

## Common Flag Patterns

### command stackit submit --no-interactive
- `--stack` - Submit entire stack (alias: `ss`)
- `--draft` - Create as draft PRs
- `--edit` - Edit PR metadata interactively

**Examples:**
```bash
# Submit current branch and ancestors
command stackit submit --no-interactive

# Submit entire stack
command stackit submit --no-interactive --stack

# Submit as drafts
command stackit submit --no-interactive --draft --stack
```

### command stackit sync --no-interactive
- `--restack` - Auto-restack after cleanup

**What it does:**
1. Pulls latest from trunk/main
2. Deletes branches whose PRs were merged
3. Deletes branches whose PRs were closed
4. Optionally restacks remaining branches

### command stackit foreach --no-interactive
**Usage:** `command stackit foreach --no-interactive "command to run"`

**Examples:**
```bash
# Build all branches (use project's build command from README.md)
command stackit foreach --no-interactive "<build-command>"

# Test all branches (use project's test command from README.md)
command stackit foreach --no-interactive "<test-command>"

# Show status on each
command stackit foreach --no-interactive "git status --short"
```

## Workflow Examples

### Start a feature stack
```bash
git add .
echo "feat: implement user authentication" | command stackit create --no-interactive

# Add tests to the same branch (branches can have multiple commits)
git add .
git commit -m "test: add auth tests"

# Work on next part as a separate stacked branch
git add .
echo "feat: add JWT token validation" | command stackit create --no-interactive
```

### Submit for review
```bash
command stackit submit --no-interactive --stack
```

### After code review changes
```bash
git add .
command stackit modify --no-interactive
# Propagate the amend to descendants of this branch only
command stackit restack --branch $(git branch --show-current) --upstack --no-interactive
command stackit submit --no-interactive
```

### Restack scope cheat sheet

Prefer the narrowest scope that covers what actually changed:

| Situation | Command |
|-----------|---------|
| Amended/modified one branch | `command stackit restack --branch <branch> --upstack --no-interactive` |
| Uncertain which branches need restack (single stack) | `command stackit restack --branch <stack-root> --upstack --no-interactive` |
| Multiple independent stacks need restack (post-sync, shared parent change) | `command stackit restack --all-stacks --continue-on-conflict --no-interactive` |
| Specific set of independent roots | `command stackit restack --stacks <root1>,<root2> --continue-on-conflict --no-interactive` |

Use `--json` for programmatic runs; it reports which branches were restacked, skipped, or conflicted so you can skip a redundant follow-up pass.

### Sync with main
```bash
command stackit sync --no-interactive --restack
```
