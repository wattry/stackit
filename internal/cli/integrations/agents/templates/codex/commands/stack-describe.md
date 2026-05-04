---
description: Use when the user wants stack or PR descriptions generated or refreshed from commit history. Trigger phrases include "describe the stack", "generate PR descriptions", and "update the stack description".
---

# Stack Describe

Generate or refresh stack and PR descriptions from the current branch history.

## Workflow

1. Inspect:

   ```bash
   stackit log --no-interactive
   git log --oneline --decorate -20
   ```

2. Read PR template if present:

   ```bash
   git ls-files .github/pull_request_template.md CONTRIBUTING.md
   ```

3. Run the Stackit description command available in this repo:

   ```bash
   stackit describe --no-interactive
   ```

4. Report which descriptions were generated or updated.

Descriptions should include a concrete summary and test plan. Do not use placeholders.
