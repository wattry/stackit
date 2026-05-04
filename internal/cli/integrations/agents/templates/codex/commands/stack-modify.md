---
description: Use when the user wants to amend the current stacked branch's commit. Trigger phrases include "amend this", "fix the current commit", "modify the branch", and "add this to the current commit". Runs `stackit modify`.
---

# Stack Modify

Amend the current stacked branch or add a follow-up commit to it.

## Workflow

1. Inspect:

   ```bash
   git status --short
   git diff --stat
   git diff --cached --stat
   stackit log --no-interactive
   git log --oneline -5
   ```

2. If there are no changes, stop.

3. Stage changes unless the user explicitly requested staged-only behavior:

   ```bash
   git add -A
   ```

4. Amend by default:

   ```bash
   stackit modify --no-interactive --no-edit
   ```

   If a new message is needed:

   ```bash
   stackit modify --no-interactive -m "<message>"
   ```

   If the user asked for a new commit on this branch:

   ```bash
   stackit modify --no-interactive -c -m "<message>"
   ```

5. Verify:

   ```bash
   stackit log --no-interactive
   ```

Never use `git commit --amend`; `stackit modify` handles stack metadata.
