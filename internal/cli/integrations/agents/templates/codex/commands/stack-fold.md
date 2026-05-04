---
description: Use when granular branches should be folded into their parent. Trigger phrases include "fold this branch", "squash into parent", and "merge this branch into the previous one". Runs `stackit fold`.
---

# Stack Fold

Fold a branch into its parent when it is too small to review separately.

## Workflow

1. Inspect:

   ```bash
   git status --short
   stackit log --no-interactive
   ```

2. Confirm the target branch and parent with the user, because folding rewrites stack structure.

3. Run the fold command appropriate for the current Stackit CLI:

   ```bash
   stackit fold --no-interactive
   ```

4. Restack descendants if Stackit reports they need it:

   ```bash
   stackit restack --upstack --no-interactive
   ```

5. Verify with `stackit log --no-interactive`.
