---
description: Sync with trunk and cleanup merged branches
model: haiku
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
---

# Stack Sync

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Task

Sync the stack with trunk: pull latest, cleanup merged branches, restack.

1. Run `command stackit sync --dry-run --json --restack --no-interactive` to preview cleanup and restack work.
2. Parse `would_clean`, `would_restack`, `would_restack_stacks`, and `skipped_stacks` from the JSON preview.
3. If ALL branches would be deleted, confirm with user first using `AskUserQuestion`.
4. Run `command stackit sync --no-restack --no-interactive` so cleanup happens once and restack can use the narrowest safe scope.
5. If branches remain and `would_restack` was non-empty, choose the restack scope:
   - If `would_restack_stacks` has exactly one root, run `command stackit restack --branch <that-root> --upstack --no-interactive`.
   - If `would_restack_stacks` lists several roots, run `command stackit restack --stacks <root-a>,<root-b> --continue-on-conflict --no-interactive`.
   - If every remaining independent stack needs restack or the root list is unavailable, run `command stackit restack --all-stacks --continue-on-conflict --no-interactive`.
6. Show final state with `command stackit log --json --no-interactive`.

If `--continue-on-conflict` reports skipped conflicts, no rebase is active for those skipped branches. To resolve one, run `command stackit restack --branch <conflicted-branch> --upstack --no-interactive`, then resolve conflicts and run `command stackit continue`.

You can call multiple tools in a single response.

## Follow-up

After sync completes, if branches remain, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Stack synced. What would you like to do next?"
- Options:
  - "Submit updates (Recommended)" → Invoke `/stack-submit` using Skill tool
  - "Done for now" → End with summary
