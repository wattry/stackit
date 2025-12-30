---
description: Diagnose and fix common stack issues
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Fix

Diagnose and automatically fix common stack problems.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short 2>/dev/null`

## Instructions

1. **Analyze stack health first**:
   ```bash
   bash ~/.claude/skills/stackit/scripts/analyze_stack.sh
   ```
   This identifies issues and suggests fixes

2. Review `stackit log --no-interactive` for structural issues and branch relationships

3. Check for and fix these common issues in priority order:

   **Compilation errors after absorb (PRIORITY FIX):**
   - Common after `stackit absorb` when absorbed changes depend on files/changes that didn't get cleanly absorbed
   - **Use detailed workflow:** See [../skills/stackit/workflows/fix-absorb.md](../skills/stackit/workflows/fix-absorb.md)
   - **Workflow checklist pattern:**
     ```
     Absorb Fix Progress:
     - [ ] Step 1: Identify build/test commands
     - [ ] Step 2: Build and test each branch
     - [ ] Step 3: Identify failed branches
     - [ ] Step 4: Find missing dependencies
     - [ ] Step 5: Apply fixes
     - [ ] Step 6: Verify entire stack
     ```
   - **Validation loop:** For each branch:
     - Run build/test
     - If fails → find dependency → apply fix → re-run build/test
     - If passes → mark complete → next branch
   - **Only complete when entire stack builds successfully**

   **Rebase in progress:**
   - Check `git status` for "rebase in progress"
   - If conflicts: See [../skills/stackit/workflows/conflict-resolution.md](../skills/stackit/workflows/conflict-resolution.md)
   - Guide through resolution, then `stackit continue --no-interactive`
   - If user wants to abort: `stackit abort --no-interactive`

   **Branches need restack:**
   - Run `stackit restack --no-interactive` to rebase branches
   - Handle conflicts if they occur

   **Orphaned branches (parent was merged):**
   - Run `stackit sync --no-interactive` to reparent

   **PR base mismatch:**
   - Run `stackit submit --no-interactive` to update PR bases

   **Uncommitted changes blocking operation:**
   - Suggest stashing or committing

4. **Final verification with validation loop**:
   - Run build on entire stack: `stackit foreach --no-interactive "<build-command>"`
   - If ANY branch fails:
     - Return to step 3
     - Fix the failing branch
     - Re-run verification
   - Only consider fix complete when ALL branches build successfully
   - Show updated stack state with `stackit log --no-interactive`

## Error Handling
- For unrecoverable issues: suggest `stackit undo --no-interactive --yes`
- If unsure: ask user before destructive operations
