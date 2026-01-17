---
description: Sync with trunk, cleanup merged branches, and restack
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion
---

# Stack Sync

Sync stack with remote: pull trunk, cleanup merged branches, restack.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Instructions

1. Run `command stackit sync --dry-run --no-interactive` and show user what will be deleted
2. If ALL branches would be deleted, use `AskUserQuestion`:
   - Header: "Delete all"
   - Question: "Sync will delete ALL branches in your stack. Are you sure?"
   - Options:
     - "Yes, delete all" - Proceed with full cleanup
     - "Cancel" - Abort sync
3. Run `command stackit sync --no-interactive`
4. If branches remain, run `command stackit restack --no-interactive`
5. Show final stack state

## Do NOT
- Skip the dry-run preview
- Proceed without confirmation if all branches will be deleted
