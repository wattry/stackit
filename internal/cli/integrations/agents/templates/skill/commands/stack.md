# Stack Operations Commands

Commands for managing the entire stack or multiple branches.

> **CRITICAL:** Always run these commands with `--no-interactive`.

## Stack Maintenance

| Command | Description |
|---------|-------------|
| `stackit restack --no-interactive` | Rebase all branches to ensure proper ancestry |
| `stackit sync --no-interactive` | Pull trunk, delete merged branches, restack |
| `stackit info --stack --json --no-interactive` | Export full stack metadata as JSON for analysis |
| `stackit merge` | Merge approved PRs and cleanup |
| `stackit fold --no-interactive` | Fold current branch into its parent |

## Bulk Operations

| Command | Description |
|---------|-------------|
| `stackit foreach --no-interactive` | Run command on each branch in stack |
| `stackit submit --no-interactive` | Push branches and create/update PRs |
| `stackit reorder` | Interactively reorder branches |
| `stackit move` | Rebase branch onto new parent |

## Common Flag Patterns

### stackit submit --no-interactive
- `--stack` - Submit entire stack (alias: `ss`)
- `--draft` - Create as draft PRs
- `--edit` - Edit PR metadata interactively

**Examples:**
```bash
# Submit current branch and ancestors
stackit submit --no-interactive

# Submit entire stack
stackit submit --no-interactive --stack

# Submit as drafts
stackit submit --no-interactive --draft --stack
```

### stackit sync --no-interactive
- `--restack` - Auto-restack after cleanup

**What it does:**
1. Pulls latest from trunk/main
2. Deletes branches whose PRs were merged
3. Deletes branches whose PRs were closed
4. Optionally restacks remaining branches

### stackit foreach --no-interactive
**Usage:** `stackit foreach --no-interactive "command to run"`

**Examples:**
```bash
# Build all branches
stackit foreach --no-interactive "just build"

# Test all branches
stackit foreach --no-interactive "npm test"

# Show status on each
stackit foreach --no-interactive "git status --short"
```

## Workflow Examples

### Start a feature stack
```bash
git add .
echo "feat: implement user authentication" | stackit create --no-interactive
# Work on next part
git add .
echo "feat: add JWT token validation" | stackit create --no-interactive
```

### Submit for review
```bash
stackit submit --no-interactive --stack
```

### After code review changes
```bash
git add .
stackit modify --no-interactive
stackit restack --no-interactive
stackit submit --no-interactive
```

### Sync with main
```bash
stackit sync --no-interactive --restack
```
