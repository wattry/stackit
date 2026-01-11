---
description: Intelligently absorb working changes into correct commits
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Grep
---

# Stack Absorb

Absorb working directory changes into the correct commits in your stack.

## Context
- Current branch: !`git branch --show-current`
- Changed files: !`git status --short`
- Changes: !`git diff --stat | head -30`
- Stack state: !`command stackit log --no-interactive 2>&1`

## How Absorb Works

Absorb assigns each change to the commit that last modified those lines. This is usually correct but can cause build errors when dependencies split across commits.

## Instructions

1. Show user what will be absorbed (from context above)
2. Explain the risk: changes may split across commits causing build errors
3. Get confirmation (absorb modifies git history)
4. Run `command stackit absorb --no-interactive --force`
5. **Validate the stack builds**:
   - Find the build command (check README, justfile, package.json)
   - Run `command stackit foreach --no-interactive "<build-command>"`
6. If all builds pass: done
7. If builds fail:
   - Identify which branches fail and why
   - Fix dependencies (move code between commits)
   - Re-run validation
   - After 2 failed attempts, suggest `command stackit undo --yes` and manual commits instead

## Do NOT
- Skip build validation after absorb
- Loop indefinitely on failed fixes (max 2 attempts, then undo)
- Leave the stack in a broken state
