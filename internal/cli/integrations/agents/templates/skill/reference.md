# Stackit Command Reference

Quick reference for all stackit commands. For detailed documentation, see:
- **Navigation details:** [commands/navigation.md](commands/navigation.md)
- **Branch operation details:** [commands/branch.md](commands/branch.md)
- **Stack operation details:** [commands/stack.md](commands/stack.md)
- **Recovery details:** [commands/recovery.md](commands/recovery.md)

> **CRITICAL:** Always run these commands with `--no-interactive`. For commands that require confirmation, include `--force` (for absorb) or `--yes` (for undo/merge).

## Utility Scripts

Run these helper scripts for analysis:

```bash
# Analyze stack health and get suggestions
bash ~/.claude/skills/stackit/scripts/analyze_stack.sh
```

## Navigation Commands

| Command | Description |
|---------|-------------|
| `stackit log` | Display the branch tree visualization |
| `stackit log full` | Show tree with GitHub PR status and CI checks |
| `stackit checkout [branch]` | Switch to a specific branch |
| `stackit up` | Move to the child branch |
| `stackit down` | Move to the parent branch |
| `stackit top` | Move to the top of the stack |
| `stackit bottom` | Move to the bottom of the stack |
| `stackit trunk` | Return to the main/trunk branch |
| `stackit children` | Show children of current branch |
| `stackit parent` | Show parent of current branch |

## Branch Management

| Command | Description |
|---------|-------------|
| `stackit create --no-interactive [name]` | Create a new stacked branch |
| `stackit modify --no-interactive` | Amend current commit (like git commit --amend) |
| `stackit absorb` | Auto-amend changes to correct commits in stack |
| `stackit split` | Split current branch into multiple branches |
| `stackit squash` | Squash all commits on current branch |
| `stackit fold` | Merge current branch into its parent |
| `stackit pop` | Delete branch but keep changes in working tree |
| `stackit delete` | Delete current branch and metadata |
| `stackit rename [name]` | Rename current branch |
| `stackit scope [name]` | Manage logical scope (Jira/Linear ID) |

## Stack Operations

| Command | Description |
|---------|-------------|
| `stackit restack --no-interactive` | Rebase all branches to ensure proper ancestry |
| `stackit foreach` | Run command on each branch in stack |
| `stackit submit --no-interactive` | Push branches and create/update PRs |
| `stackit sync --no-interactive` | Pull trunk, delete merged branches, restack |
| `stackit merge` | Merge approved PRs and cleanup |
| `stackit reorder` | Interactively reorder branches |
| `stackit move` | Rebase branch onto new parent |

## Recovery & Utilities

| Command | Description |
|---------|-------------|
| `stackit undo` | Restore repo to state before a command |
| `stackit continue` | Continue interrupted operation |
| `stackit abort` | Abort interrupted operation |
| `stackit doctor` | Diagnose and fix setup issues |
| `stackit info` | Show detailed branch info |
| `stackit track` | Start tracking a branch |
| `stackit untrack` | Stop tracking a branch |
| `stackit debug` | Dump debugging info |

## Common Flag Patterns

### stackit create --no-interactive
- `-m "message"` - Commit message
- `--all` - Stage all changes first
- `--insert` - Insert between current and child

### stackit submit --no-interactive
- `--stack` - Submit entire stack (alias: `ss`)
- `--draft` - Create as draft PRs
- `--edit` - Edit PR metadata interactively

### stackit sync --no-interactive
- `--restack` - Auto-restack after cleanup

## Workflow Examples

### Start a new feature
```bash
git add .
echo "feat: add new feature" | stackit create --no-interactive
```

### Stack another change
```bash
git add .
echo "feat: extend feature" | stackit create --no-interactive
```

### Add more commits to current branch
```bash
# A stacked branch can have multiple commits - no need to create a new branch
git add .
git commit -m "test: add tests for feature"
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

## Troubleshooting

For detailed troubleshooting workflows, see:
- **Fixing absorb errors:** [workflows/fix-absorb.md](workflows/fix-absorb.md)
- **Conflict resolution:** [workflows/conflict-resolution.md](workflows/conflict-resolution.md)
- **Recovery commands:** [commands/recovery.md](commands/recovery.md)

### Quick Fixes

| Issue | Solution |
|-------|----------|
| "Branch needs restack" | `stackit restack --no-interactive` |
| "Rebase conflict" | Resolve conflicts, `git add <files>`, `stackit continue` |
| "Orphaned branch" | `stackit sync --no-interactive` to reparent |
| "PR base mismatch" | `stackit submit --no-interactive` to update PRs |
| Build breaks after absorb | See [workflows/fix-absorb.md](workflows/fix-absorb.md) |
