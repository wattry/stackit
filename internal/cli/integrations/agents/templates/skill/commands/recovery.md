# Recovery & Utilities Commands

Commands for troubleshooting, recovery, and diagnostics.

> **CRITICAL:** Always run these commands with `--no-interactive`. For commands that require confirmation, include `--force` (for absorb) or `--yes` (for undo/merge).

## Recovery Commands

| Command | Description |
|---------|-------------|
| `stackit undo --no-interactive --yes` | Restore repo to state before a command |
| `stackit continue --no-interactive` | Continue interrupted operation |
| `stackit abort --no-interactive` | Abort interrupted operation |

## Diagnostic Commands

| Command | Description |
|---------|-------------|
| `stackit doctor --no-interactive` | Diagnose and fix setup issues |
| `stackit info --no-interactive` | Show detailed branch info |
| `stackit debug --no-interactive` | Dump debugging info |

## Recovery Workflows

### Rebase in Progress

If you see "rebase in progress":
```bash
# Check status
git status

# If conflicts exist:
# 1. Resolve conflicts in files
# 2. Stage resolved files
git add <resolved-files>

# 3. Continue
stackit continue --no-interactive

# Or abort if needed
stackit abort --no-interactive
```

### Undo Last Operation

```bash
# Restore to previous state
stackit undo --no-interactive --yes

# Check if it worked
stackit log --no-interactive
```

### Branch Needs Restack

```bash
# Rebase all branches to fix ancestry
stackit restack --no-interactive

# Handle conflicts if they occur
# Then continue
stackit continue --no-interactive
```

### Orphaned Branch (Parent Merged)

```bash
# Sync will reparent automatically
stackit sync --no-interactive

# Or manually restack
stackit restack --no-interactive
```

### PR Base Mismatch

```bash
# Submit will update PR base branches
stackit submit --no-interactive
```

### Uncommitted Changes Blocking Operation

```bash
# Option 1: Commit them
git add .
stackit modify --no-interactive

# Option 2: Stash them
git stash

# Do operation, then restore
git stash pop
```

## Diagnostic Workflows

### Check Stack Health

```bash
# View structure
stackit log --no-interactive

# Get detailed info
stackit info --no-interactive

# Check for issues
stackit doctor --no-interactive
```

### Debug Issues

```bash
# Dump all diagnostic info
stackit debug --no-interactive

# This shows:
# - Git refs
# - Stackit metadata
# - Branch relationships
# - Configuration
```

## Troubleshooting Checklist

When things go wrong:
1. Check status: `git status` and `stackit log --no-interactive`
2. Look for interrupted operations (rebase, merge)
3. Try `stackit doctor --no-interactive` for automatic fixes
4. If stuck, `stackit abort --no-interactive` then `stackit undo --no-interactive --yes`
5. For persistent issues, check `stackit debug --no-interactive` output
