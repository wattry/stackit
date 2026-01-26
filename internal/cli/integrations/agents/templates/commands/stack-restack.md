---
description: Rebase all branches to ensure proper ancestry
model: claude-haiku-4-20250414
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
---

# Stack Restack

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Task

Rebase all branches in the stack to ensure proper parent-child ancestry.

**Preconditions** (check context above):
- Must be on a branch (not detached HEAD)
- Must have clean working directory (no uncommitted changes)

If preconditions fail, inform user and stop.

Otherwise, run `command stackit restack --no-interactive` and show the result.

If conflicts occur, inform user they need to resolve conflicts and run `command stackit continue`.

## Follow-up

After successful restack, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Stack rebased. What would you like to do next?"
- Options:
  - "Submit to update PRs (Recommended)" → Invoke `/stack-submit` using Skill tool
  - "Done for now" → End with summary
