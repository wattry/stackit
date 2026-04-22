---
description: Rebase all branches to ensure proper ancestry
model: haiku
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
---

# Stack Restack

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Task

Rebase all branches in the stack to ensure proper parent-child ancestry.

**Preconditions** (check context above):
- Must be on a branch (not detached HEAD)
- Must have clean working directory (no uncommitted changes)

If preconditions fail, inform user and stop.

Otherwise, choose the narrowest safe restack scope and show the result:
- If a specific branch caused the issue or was just amended, run `command stackit restack --branch <branch> --upstack --no-interactive`.
- If a whole independent stack needs restack, run `command stackit restack --branch <stack-root> --upstack --no-interactive`.
- If multiple independent stacks need restack, run one scoped restack per affected stack root.
- Only run `command stackit restack --no-interactive` when the user explicitly wants the current stack restacked and no narrower branch/root is known.

If conflicts occur, inform user they need to resolve conflicts and run `command stackit continue`.

## Follow-up

After successful restack, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Stack rebased. What would you like to do next?"
- Options:
  - "Submit to update PRs (Recommended)" → Invoke `/stack-submit` using Skill tool
  - "Done for now" → End with summary
