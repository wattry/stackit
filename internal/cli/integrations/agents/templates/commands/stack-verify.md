---
description: Verify stack health by running checks on all branches
allowed-tools: Bash(stackit:*), Bash(git:*)
argument-hint: [check-command]
---

# Stack Verify

Run verification checks on all branches in the stack and report results.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Arguments
$ARGUMENTS

## Instructions

### 1. Determine the check command

If provided in arguments, use that. Otherwise:
1. Check README.md or CONTRIBUTING.md for build/test instructions
2. Look for common build files (Makefile, package.json scripts, etc.)
3. If not found, ask the user: "What command should I use to verify the code?"

### 2. Run verification

First, go to bottom of stack to ensure consistent ordering:
```bash
command stackit bottom --no-interactive
```

Then run checks:

**Quick mode (default)** - stop at first failure:
```bash
command stackit foreach --upstack "<check-command>" 2>&1
```

**Full diagnostic mode** - check all branches:
```bash
command stackit foreach --upstack --no-fail-fast "<check-command>" 2>&1
```

Use quick mode unless the user wants to see ALL failures.

### 3. Parse foreach output

Foreach output looks like:
```
Running on branch-1...
✓ branch-1 (exit 0)
Running on branch-2...
✓ branch-2 (exit 0)
Running on branch-3...
✗ branch-3 (exit 1)
  internal/foo.go:42: undefined: someVar
```

Look for `✗` or non-zero exit codes to identify failures.

### 4. Present results clearly

**Quick mode format** (stops at first failure):
```
Stack Verification Results
==========================
✓ branch-name-1 - passed
✓ branch-name-2 - passed
✗ branch-name-3 - FAILED

Summary: 2 passed, 1 failed (stopped at first failure)
First failure: branch-name-3
```

**Full diagnostic format** (checks all):
```
Stack Verification Results
==========================
✓ branch-name-1 - passed
✓ branch-name-2 - passed
✗ branch-name-3 - FAILED (first failure)
✗ branch-name-4 - FAILED

Summary: 2 passed, 2 failed
First failure: branch-name-3
```

### 5. Recommend next steps

- **All pass**: "Stack is healthy and ready for submit"
- **Failures found**: "Run /stack-fix to fix issues. Start by fixing branch-name-3, then restack to propagate the fix to dependent branches."

## Edge Cases

**Uncommitted changes**: Warn user - foreach may fail to checkout branches.

**Not on tracked branch**: Report that verification requires a tracked branch.

**All branches pass**: Clearly report success, suggest `/stack-submit`.

**Check command fails immediately**: Suggest verifying command works manually first.

## Do NOT
- Attempt to fix any issues (that's /stack-fix's job)
- Skip identifying the first-failure branch
- Modify any files or make commits
- Use git checkout directly
