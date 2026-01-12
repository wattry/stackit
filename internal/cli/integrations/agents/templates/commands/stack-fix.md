---
description: Diagnose and fix common stack issues
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Edit, Glob, Grep
---

# Stack Fix

Diagnose and fix stack problems, including build/lint/test failures.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Instructions

### 1. Identify the Issue Type

Check the context and look for these indicators:

**Rebase in progress** (git status shows "rebase in progress" or "interactive rebase"):
- Guide user through conflict resolution
- Then `command stackit continue` or `command stackit abort`

**Build/lint/test failures** (user reports build errors, or you see compiler errors):
- Follow the Build Failure Workflow below

**Branches need restack** (stackit log shows "needs restack" or branches are out of sync):
- Run `command stackit restack --no-interactive`

**Orphaned branches** (stackit log shows branch with no parent, or parent was merged):
- Run `command stackit sync --no-interactive`

**Uncommitted changes** (git status shows modified/untracked files):
- Ask user to commit or stash before proceeding

**Not on tracked branch** (current branch not shown in stackit log with ◉):
- Guide user to checkout a tracked branch first

### 2. Build Failure Workflow

#### Step 0: Pre-checks
Before running foreach, verify:
- No uncommitted changes (check git status from context)
- No rebase in progress
- Currently on a tracked branch

If uncommitted changes exist, ask user to commit or stash first.

#### Step 1: Determine the check command
If not obvious from context, ask what command verifies their code:
- Go: `go build ./...` or `just check`
- Node: `npm test` or `npm run build`
- Rust: `cargo build`
- Python: `pytest` or `make test`

Optionally, verify the command works on current branch first:
```bash
<check-command>
```

#### Step 2: Go to bottom of stack and run checks upward
```bash
command stackit bottom --no-interactive
command stackit foreach --upstack "<check-command>" 2>&1
```

This starts at the bottom branch (closest to trunk) and walks toward leaves, stopping at the first failure. The failing branch is where the bug was introduced.

**Example foreach output:**
```
Running on branch-1...
✓ branch-1 (exit 0)
Running on branch-2...
✗ branch-2 (exit 1)
  Error: undefined: someVariable
```

Parse the output to find the FIRST branch with `✗` or non-zero exit - that's where to fix.

**If all branches show `✓`**: Report success! Stack is healthy. Skip to "After Fixes".

#### Step 3: Checkout the failing branch
```bash
command stackit checkout <failing-branch> --no-interactive
```

#### Step 4: Fix the issue
- Read the error output to understand the problem
- Make the necessary code changes
- Stage and commit:
  ```bash
  git add -A
  git commit -m "fix: <description>"
  ```

#### Step 5: Propagate the fix via restack
```bash
command stackit restack --no-interactive
```

This rebases all child branches onto the fixed branch, propagating your fix.

**If restack has conflicts**:
- Help user resolve the conflicts
- Then run `command stackit continue` to proceed
- Or `command stackit abort` to cancel

#### Step 6: Verify all branches now pass
```bash
command stackit foreach --stack "<check-command>" 2>&1
```

If it stops at another failure, repeat from Step 2 (there may be multiple independent issues).

### 3. After Fixes

Verify stack is healthy:
- `command stackit log` shows clean tree
- All branches pass checks

## Key Insight

**Fix at the SOURCE branch (first failure), then restack to propagate.**

Never fix the same issue on multiple branches - that's painful and error-prone!
The restack command automatically propagates your fix to all child branches.

## Edge Cases

**Uncommitted changes**: Check git status first, ask user to commit/stash.

**Rebase already in progress**: Use `stackit continue` or `stackit abort` before foreach.

**Not on tracked branch**: Guide user to checkout a tracked branch first.

**Branching stack (multiple children)**: foreach handles this - all descendants are checked.

**Multiple independent failures**: After fixing one, re-run foreach to find next failure.

## Do NOT
- Fix the same bug on multiple branches manually
- Use `git checkout` directly - use `command stackit checkout` instead
- Make destructive changes without user confirmation
- Loop indefinitely on fixes (max 2 attempts per issue type)
- Skip asking what check command to use if unclear
