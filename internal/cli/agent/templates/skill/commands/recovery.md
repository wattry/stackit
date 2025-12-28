# Recovery & Utilities Commands

Commands for troubleshooting, recovery, and diagnostics.

## Recovery Commands

| Command | Description |
|---------|-------------|
| `stackit undo` | Restore repo to state before a command |
| `stackit continue` | Continue interrupted operation |
| `stackit abort` | Abort interrupted operation |

## Diagnostic Commands

| Command | Description |
|---------|-------------|
| `stackit doctor` | Diagnose and fix setup issues |
| `stackit info` | Show detailed branch info |
| `stackit debug` | Dump debugging info |

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
stackit continue

# Or abort if needed
stackit abort
```

### Undo Last Operation

```bash
# Restore to previous state
stackit undo

# Check if it worked
stackit log
```

### Branch Needs Restack

```bash
# Rebase all branches to fix ancestry
stackit restack

# Handle conflicts if they occur
# Then continue
stackit continue
```

### Orphaned Branch (Parent Merged)

```bash
# Sync will reparent automatically
stackit sync

# Or manually restack
stackit restack
```

### PR Base Mismatch

```bash
# Submit will update PR base branches
stackit submit
```

### Uncommitted Changes Blocking Operation

```bash
# Option 1: Commit them
git add .
stackit modify

# Option 2: Stash them
git stash

# Do operation, then restore
git stash pop
```

## Diagnostic Workflows

### Check Stack Health

```bash
# View structure
stackit log

# Get detailed info
stackit info

# Check for issues
stackit doctor
```

### Debug Issues

```bash
# Dump all diagnostic info
stackit debug

# This shows:
# - Git refs
# - Stackit metadata
# - Branch relationships
# - Configuration
```

## Troubleshooting Checklist

When things go wrong:
1. Check status: `git status` and `stackit log`
2. Look for interrupted operations (rebase, merge)
3. Try `stackit doctor` for automatic fixes
4. If stuck, `stackit abort` then `stackit undo`
5. For persistent issues, check `stackit debug` output
