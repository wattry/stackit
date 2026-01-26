---
description: Sync with trunk and cleanup merged branches
model: claude-haiku-4-20250414
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
---

# Stack Sync

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Task

Sync the stack with trunk: pull latest, cleanup merged branches, restack.

1. Run `command stackit sync --dry-run --no-interactive` to preview
2. If ALL branches would be deleted, confirm with user first using `AskUserQuestion`
3. Run `command stackit sync --no-interactive`
4. If branches remain, run `command stackit restack --no-interactive`
5. Show final state with `command stackit log --no-interactive`

You can call multiple tools in a single response.

## Follow-up

After sync completes, if branches remain, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Stack synced. What would you like to do next?"
- Options:
  - "Submit updates (Recommended)" → Invoke `/stack-submit` using Skill tool
  - "Done for now" → End with summary
