---
description: Modify the current branch by amending or creating a new commit
model: sonnet
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
argument-hint: [-m "message"] [-a] [-c] [--no-edit]
---

# Stack Modify

## Context
- Current branch: !`git branch --show-current`
- Unstaged changes: !`git diff --stat 2>&1 | head -20`
- Staged changes: !`git diff --cached --stat 2>&1 | head -20`
- Recent commits on branch: !`git log --oneline -5 2>&1`
- Stack state: !`stackit log --no-interactive 2>&1`

## Arguments
$ARGUMENTS

## Task

Modify the current branch by amending its commit or creating a new commit. Automatically restacks descendants after modification.

**Two modes:**
- **Amend (default):** Amends the current branch's latest commit
- **New commit (`-c`):** Creates a new commit on the branch

### Steps

1. **Check for changes:**
   - If no staged changes and no unstaged changes, inform user and stop
   - If no staged changes but unstaged changes exist, run `git add --all` to stage them
   - If user provided `-a`, run `git add --all` first

2. **Determine the message:**
   - If `-m "message"` provided, use that message
   - If `--no-edit` or `-n` provided, keep the existing commit message (amend mode only)
   - If neither provided and amending, generate a message describing what changed based on the diff, then use `-m`
   - If neither provided and creating new commit (`-c`), generate a message based on the diff

3. **Run the command:**
   ```bash
   # Amend (default) with message
   stackit modify -m "<message>"

   # Amend keeping existing message
   stackit modify --no-edit

   # New commit
   stackit modify -c -m "<message>"

   # Stage all + amend
   stackit modify -a -m "<message>"
   ```

4. **Handle results:**
   - On success, report what was modified and that descendants were restacked
   - On failure, report the error clearly

**Never use:** `git commit --amend` — always use `stackit modify` so descendants are restacked.

## Follow-up

After successful modification, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Branch modified. What would you like to do next?"
- Options:
  - "Submit to update PRs (Recommended)" → Invoke `/stack-submit` using Skill tool
  - "Continue working" → Tell user to make more changes
  - "Done for now" → End with summary
