---
description: Verify stack health by running checks on all branches
model: claude-sonnet-4-20250514
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill, Task
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
3. If not found, use `AskUserQuestion`:
   - Header: "Check command"
   - Question: "What command should I use to verify the code?"
   - Options:
     - "Skip verification" - Don't run checks
     - "Let me specify" - I'll provide the command

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

**Parallel mode** - For large stacks (5+ branches) with independent branches, consider spawning parallel Task subagents to verify sibling branches simultaneously. Only use this when branches don't have linear dependencies.

If uncertain which mode to use, prompt with `AskUserQuestion`:
- Header: "Verify mode"
- Question: "How should I run verification?"
- Options:
  - "Quick (Recommended)" - Stop at first failure
  - "Full diagnostic" - Check all branches, report all failures

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

## Tool Trust

Trust all tools work without error. Don't run exploratory commands to verify tool behavior.

## Do NOT
- Attempt to fix any issues (that's /stack-fix's job)
- Skip identifying the first-failure branch
- Modify any files or make commits
- Use git checkout directly

## Follow-up

After verification completes, use `AskUserQuestion` based on results:

**If all branches pass:**
- Header: "Next step"
- Question: "All branches pass verification. What would you like to do next?"
- Options:
  - label: "Submit PRs (Recommended)"
    description: "Push branches and create/update pull requests"
  - label: "View stack"
    description: "Show current stack state"
  - label: "Done for now"
    description: "No follow-up action needed"

**If failures found:**
- Header: "Next step"
- Question: "Verification found failures. What would you like to do next?"
- Options:
  - label: "Fix issues (Recommended)"
    description: "Diagnose and fix the failing branches"
  - label: "View details"
    description: "Show more information about failures"
  - label: "Done for now"
    description: "I'll fix manually"

Based on response:
- **"Submit PRs"**: Invoke `/stack-submit` skill using the `Skill` tool
- **"Fix issues"**: Invoke `/stack-fix` skill using the `Skill` tool
- **"View stack"** / **"View details"**: Run `command stackit log --no-interactive`
- **"Done for now"**: End with summary of verification results
