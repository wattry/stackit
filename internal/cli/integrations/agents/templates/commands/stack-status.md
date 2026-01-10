---
description: View current stack state, branch position, and health status
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Status

Show the current stack state and identify any issues.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`stackit log --no-interactive 2>&1`
- Branch info: !`stackit info --json --no-interactive 2>&1`

## Instructions

Based on the context above:
- If branches need restack: suggest `stackit restack`
- If branches have no PR: suggest `stackit submit`
- If uncommitted changes: note them
- If stack is healthy: confirm status is good
