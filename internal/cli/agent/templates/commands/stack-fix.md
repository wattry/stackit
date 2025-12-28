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

1. Run `stackit log` to identify structural issues and branch relationships

2. Check for and fix these common issues:

   **Compilation errors after absorb (priority fix):**
   - Common after `stackit absorb` when absorbed changes depend on files/changes that didn't get cleanly absorbed
   - Check README.md and CONTRIBUTING.md for build/test/lint commands
   - For each branch in stack (bottom to top):
     - Checkout the branch: `git checkout <branch>`
     - Run project's build command
     - Run project's test command
     - If failures occur:
       - Analyze error messages for missing files/functions/types
       - Check upstack branches for those changes: `git diff <branch>..<child-branch>`
       - If needed changes found in upstack branches:
         - Cherry-pick or apply specific changes: `git cherry-pick <commit>`
         - Or manually copy needed files/changes
       - Re-run build/test to verify fix
   - Continue until all branches build/test successfully

   **Rebase in progress:**
   - Check `git status` for "rebase in progress"
   - If conflicts: help user resolve, then `stackit continue`
   - If user wants to abort: `stackit abort`

   **Branches need restack:**
   - Run `stackit restack` to rebase branches
   - Handle conflicts if they occur

   **Orphaned branches (parent was merged):**
   - Run `stackit sync` to reparent

   **PR base mismatch:**
   - Run `stackit submit` to update PR bases

   **Uncommitted changes blocking operation:**
   - Suggest stashing or committing

3. After fixes, verify the entire stack builds:
   - Run `stackit foreach "<build-command>"` to build all branches
   - Show updated stack state with `stackit log`

## Error Handling
- For unrecoverable issues: suggest `stackit undo`
- If unsure: ask user before destructive operations
