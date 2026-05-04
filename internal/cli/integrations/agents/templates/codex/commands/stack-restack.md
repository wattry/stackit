---
description: Use when the stack needs to be rebased to restore proper ancestry. Trigger phrases include "restack", "rebase the stack", "my stack is out of sync", and "fix parent relationships". Runs `stackit restack`.
---

# Stack Restack

Rebase stack branches according to Stackit metadata.

## Workflow

1. Inspect:

   ```bash
   git status --short
   stackit log --no-interactive
   ```

2. For the current stack:

   ```bash
   stackit restack --upstack --no-interactive
   ```

3. For a specific root:

   ```bash
   stackit restack --branch <root> --upstack --no-interactive
   ```

4. Restack all stacks only if explicitly requested:

   ```bash
   stackit restack --all-stacks --continue-on-conflict --no-interactive
   ```

5. If conflicts occur, resolve files and use `stack-resolve`.

6. Verify with `stackit log --no-interactive`.

Do not use raw `git rebase` for stack branches.
