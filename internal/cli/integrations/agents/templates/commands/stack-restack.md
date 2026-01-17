---
description: Rebase all branches to ensure proper ancestry
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
---

# Stack Restack

Rebase all branches in the stack to ensure proper parent-child ancestry.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Instructions

1. Check preconditions from context above:
   - If no current branch (detached HEAD): suggest `command stackit checkout <branch>`
   - If uncommitted changes: suggest committing or stashing first
2. Run `command stackit restack --no-interactive`
3. If conflicts occur, guide user through resolution then `command stackit continue`
4. Show final stack state
5. Suggest `command stackit submit` to update PRs

## Do NOT
- Proceed with uncommitted changes
- Proceed in detached HEAD state
- Force through conflicts without user resolution

## Follow-up

After successful restack, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Stack rebased successfully. What would you like to do next?"
- Options:
  - label: "Submit to update PRs (Recommended)"
    description: "Push rebased branches to update remote PRs"
  - label: "View stack"
    description: "Show current stack state"
  - label: "Done for now"
    description: "No follow-up action needed"

Based on response:
- **"Submit to update PRs"**: Invoke `/stack-submit` skill using the `Skill` tool
- **"View stack"**: Run `command stackit log --no-interactive`
- **"Done for now"**: End with summary of restacked branches
