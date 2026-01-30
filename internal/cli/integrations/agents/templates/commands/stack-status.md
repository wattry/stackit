---
description: View current stack state and health
model: haiku
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Status

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`
- Branch info: !`command stackit info --json --no-interactive 2>&1`

## Task

Summarize the stack state from the context above. Note any issues:
- Branches needing restack
- Branches without PRs
- Uncommitted changes
- Detached HEAD state

If healthy, confirm the stack is in good state. Do not run additional commands.
