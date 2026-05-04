---
description: Use when the user wants to run a build or test command across every branch in the stack. Trigger phrases include "verify each branch", "check every branch builds", and "run tests on the whole stack".
---

# Stack Verify

Run verification across stack branches.

## Workflow

1. Inspect:

   ```bash
   stackit log --no-interactive
   git status --short
   ```

2. Determine check command from user input, project docs, or common files. Prefer `mise run check` when available.

3. Run across the current stack:

   ```bash
   stackit foreach --stack "<check-command>"
   ```

   For current branch and descendants:

   ```bash
   stackit foreach --upstack "<check-command>"
   ```

4. Report the first failing branch and relevant output. If all pass, report that clearly.
