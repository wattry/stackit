---
description: View current stack state, branch position, and health status
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Status

Show the current stack state and identify any issues.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`
- Branch info: !`command stackit info --json --no-interactive 2>&1`

## Instructions

Based on the context above:
- If branches need restack: suggest `command stackit restack`
- If branches have no PR: suggest `command stackit submit`
- If uncommitted changes: note them
- If stack is healthy: confirm status is good
