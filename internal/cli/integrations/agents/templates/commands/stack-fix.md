---
description: Diagnose and fix common stack issues
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Fix

Diagnose and fix common stack problems.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Instructions

1. Analyze the context to identify issues:

   **Rebase in progress**:
   - Guide user through conflict resolution
   - Then `command stackit continue` or `command stackit abort`

   **Build errors after absorb**:
   - Run build on each branch: `command stackit foreach "<build-command>"`
   - Find and fix dependency issues
   - After 2 failed attempts, suggest `command stackit undo --yes`

   **Branches need restack**:
   - Run `command stackit restack --no-interactive`

   **Orphaned branches (parent was merged)**:
   - Run `command stackit sync --no-interactive`

   **Uncommitted changes blocking operation**:
   - Suggest commit or stash

2. After fixes, verify stack is healthy:
   - `command stackit log` shows clean tree
   - All branches build successfully

## Do NOT
- Make destructive changes without user confirmation
- Loop indefinitely on fixes (max 2 attempts per issue)
