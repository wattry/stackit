---
description: Use when the user wants to sync the stack with trunk and clean up merged branches. Trigger phrases include "sync with main", "pull trunk", "clean up merged branches", and "update from origin". Runs `stackit sync`.
---

# Stack Sync

Sync with trunk and clean up branches that have landed.

## Workflow

1. Inspect:

   ```bash
   git status --short
   stackit log --no-interactive
   ```

2. If there are uncommitted changes, stop and ask whether to stash, commit, or abort.

3. Run:

   ```bash
   stackit sync --no-interactive
   ```

4. If Stackit reports branches needing restack:

   ```bash
   stackit restack --upstack --no-interactive
   ```

5. Verify:

   ```bash
   stackit log --no-interactive
   ```
