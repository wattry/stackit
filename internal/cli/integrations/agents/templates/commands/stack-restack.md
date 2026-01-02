---
description: Rebase all branches to ensure proper ancestry
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Restack

Rebase all branches in the stack to ensure proper parent-child ancestry.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log --no-interactive 2>/dev/null`
- Git status: !`git status --short 2>/dev/null`

## Instructions

1. Check for uncommitted changes with `git status`
   - If dirty: warn user and suggest committing or stashing

2. Show current stack state with `stackit log --no-interactive`

3. Run `stackit restack --no-interactive` to rebase all branches

4. **If conflicts occur:**
   - Show which files have conflicts
   - Provide guidance on resolving:
     ```
     1. Open conflicted files and resolve markers
     2. git add <resolved-files>
     3. stackit continue --no-interactive
     ```
   - Wait for user to resolve before continuing

5. **If restack succeeds:**
   - Show updated stack state
   - Inform user branches are now properly stacked

6. Suggest `stackit submit --no-interactive` to update PRs with rebased commits

## Error Handling
- If rebase fails completely: suggest `stackit abort --no-interactive` then `stackit undo --no-interactive --yes`
- If specific branch fails: show which branch and why
