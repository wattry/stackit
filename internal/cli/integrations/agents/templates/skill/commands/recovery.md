# Recovery & Utilities Commands

Commands for troubleshooting, recovery, and diagnostics.

> **CRITICAL:** Always run these commands with `command stackit ... --no-interactive`. For commands that require confirmation, include `--force` (for absorb) or `--yes` (for undo/merge).

## Recovery Commands

| Command | Description |
|---------|-------------|
| `command stackit undo --no-interactive --yes` | Restore repo to state before a command |
| `command stackit continue --no-interactive` | Continue interrupted operation |
| `command stackit abort --no-interactive` | Abort interrupted operation |

## Diagnostic Commands

| Command | Description |
|---------|-------------|
| `command stackit doctor --no-interactive` | Diagnose and fix setup issues |
| `command stackit info --no-interactive` | Show detailed branch info |
| `command stackit debug --no-interactive` | Dump debugging info |

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
command stackit continue --no-interactive

# Or abort if needed
command stackit abort --no-interactive
```

### Undo Last Operation

```bash
# Restore to previous state
command stackit undo --no-interactive --yes

# Check if it worked
command stackit log --no-interactive
```

### Branch Needs Restack

Prefer the narrowest scope that covers the affected branches:

```bash
# Most common: restack the flagged branch and its descendants
command stackit restack --branch <branch> --upstack --no-interactive

# Multiple independent stacks need attention (e.g. after sync reparented roots)
command stackit restack --all-stacks --continue-on-conflict --no-interactive

# Handle conflicts if they occur, then continue
command stackit continue --no-interactive
```

Use `--json` to see which branches were restacked vs skipped and avoid a redundant second pass.

### Orphaned Branch (Parent Merged)

```bash
# Sync will reparent automatically
command stackit sync --no-interactive

# If restack is still needed, scope it — avoid broad restacks
command stackit restack --branch <reparented-branch> --upstack --no-interactive
# Or, if sync reparented roots across multiple stacks:
command stackit restack --all-stacks --continue-on-conflict --no-interactive
```

### PR Base Mismatch

```bash
# Submit will update PR base branches
command stackit submit --no-interactive
```

### Uncommitted Changes Blocking Operation

```bash
# Option 1: Commit them
git add .
command stackit modify --no-interactive

# Option 2: Stash them
git stash

# Do operation, then restore
git stash pop
```

## Diagnostic Workflows

### Check Stack Health

```bash
# View structure
command stackit log --no-interactive

# Get detailed info
command stackit info --no-interactive

# Check for issues
command stackit doctor --no-interactive
```

### Debug Issues

```bash
# Dump all diagnostic info
command stackit debug --no-interactive

# This shows:
# - Git refs
# - Stackit metadata
# - Branch relationships
# - Configuration
```

## Troubleshooting Checklist

When things go wrong:
1. Check status: `git status` and `command stackit log --no-interactive`
2. Look for interrupted operations (rebase, merge)
3. Try `command stackit doctor --no-interactive` for automatic fixes
4. If stuck, `command stackit abort --no-interactive` then `command stackit undo --no-interactive --yes`
5. For persistent issues, check `command stackit debug --no-interactive` output
