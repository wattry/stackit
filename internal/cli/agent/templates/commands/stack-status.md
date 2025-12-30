---
description: View current stack state, branch position, and health status
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Status

Show the current stack state and identify any issues.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log --no-interactive 2>/dev/null || echo 'Stackit not initialized'`

## Instructions

1. Run `stackit log --no-interactive` to display the branch tree
2. Run `stackit info --no-interactive` to show current branch details
3. Based on the output, suggest next actions:
   - If branches need restack: suggest `stackit restack --no-interactive`
   - If branches have no PR: suggest `stackit submit --no-interactive`
   - If stack is healthy: confirm status is good

## Error Handling
- If not in a git repo: inform user
- If stackit not initialized: suggest `stackit init --no-interactive`
