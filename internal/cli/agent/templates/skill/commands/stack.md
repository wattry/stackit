# Stack Operations Commands

Commands for managing the entire stack or multiple branches.

## Stack Maintenance

| Command | Description |
|---------|-------------|
| `stackit restack` | Rebase all branches to ensure proper ancestry |
| `stackit sync` | Pull trunk, delete merged branches, restack |
| `stackit merge` | Merge approved PRs and cleanup |

## Bulk Operations

| Command | Description |
|---------|-------------|
| `stackit foreach` | Run command on each branch in stack |
| `stackit submit` | Push branches and create/update PRs |
| `stackit reorder` | Interactively reorder branches |
| `stackit move` | Rebase branch onto new parent |

## Common Flag Patterns

### stackit submit
- `--stack` - Submit entire stack (alias: `ss`)
- `--draft` - Create as draft PRs
- `--edit` - Edit PR metadata interactively

**Examples:**
```bash
# Submit current branch and ancestors
stackit submit

# Submit entire stack
stackit submit --stack

# Submit as drafts
stackit submit --draft --stack
```

### stackit sync
- `--restack` - Auto-restack after cleanup

**What it does:**
1. Pulls latest from trunk/main
2. Deletes branches whose PRs were merged
3. Deletes branches whose PRs were closed
4. Optionally restacks remaining branches

### stackit foreach
**Usage:** `stackit foreach "command to run"`

**Examples:**
```bash
# Build all branches
stackit foreach "just build"

# Test all branches
stackit foreach "npm test"

# Show status on each
stackit foreach "git status --short"
```

## Workflow Examples

### Start a feature stack
```bash
git add .
echo "feat: implement user authentication" | stackit create
# Work on next part
git add .
echo "feat: add JWT token validation" | stackit create
```

### Submit for review
```bash
stackit submit --stack
```

### After code review changes
```bash
git add .
stackit modify
stackit restack
stackit submit
```

### Sync with main
```bash
stackit sync --restack
```
