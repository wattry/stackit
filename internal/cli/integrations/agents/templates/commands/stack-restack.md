---
description: Rebase all branches to ensure proper ancestry
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Restack

Rebase all branches in the stack to ensure proper parent-child ancestry.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`stackit log --no-interactive 2>&1`

## Instructions

1. Check preconditions from context above:
   - If no current branch (detached HEAD): suggest `stackit checkout <branch>`
   - If uncommitted changes: suggest committing or stashing first
2. Run `stackit restack --no-interactive`
3. If conflicts occur, guide user through resolution then `stackit continue`
4. Show final stack state
5. Suggest `stackit submit` to update PRs

## Do NOT
- Proceed with uncommitted changes
- Proceed in detached HEAD state
- Force through conflicts without user resolution
