---
description: Use when the user wants to inspect stack health and state. Trigger phrases include "show the stack", "what's in my stack", "stack status", and "stack health". Reads `stackit log` and reports.
---

# Stack Status

Report stack state without mutating anything.

## Workflow

Run:

```bash
git status --short
stackit log --json --no-interactive
```

Summarize:

- Current branch and parent.
- Children.
- PR status when present.
- Branches needing restack.
- Failing CI or ready-to-merge signals when present.

Use plain `stackit log --no-interactive` when the user wants the visual stack.
