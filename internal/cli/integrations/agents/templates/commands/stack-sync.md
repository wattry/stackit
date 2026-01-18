---
description: Sync with trunk, cleanup merged branches, and restack
model: claude-sonnet-4-20250514
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
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

## Follow-up

After successful sync, check if branches remain and use `AskUserQuestion`:

**If branches remain with changes:**
- Header: "Next step"
- Question: "Stack synced. What would you like to do next?"
- Options:
  - label: "Submit updates (Recommended)"
    description: "Push rebased branches to update PRs"
  - label: "View stack"
    description: "Show current stack state"
  - label: "Done for now"
    description: "No follow-up action needed"

**If all branches were merged/deleted:**
- End with summary: "All branches have been merged and cleaned up. Your stack is empty."

Based on response:
- **"Submit updates"**: Invoke `/stack-submit` skill using the `Skill` tool
- **"View stack"**: Run `command stackit log --no-interactive`
- **"Done for now"**: End with summary of synced state
