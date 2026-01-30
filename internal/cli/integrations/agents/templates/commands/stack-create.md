---
description: Create a new stacked branch with commit
model: sonnet
allowed-tools: Bash(stackit:*), Bash(git:*), AskUserQuestion, Skill
argument-hint: [-m "message"] [--scope <scope>] [branch-name]
---

# Stack Create

## Context
- Current branch: !`git branch --show-current`
- Unstaged changes: !`git diff --stat 2>&1 | head -20`
- Staged changes: !`git diff --cached --stat 2>&1 | head -20`
- Recent commits (for style): !`git log --oneline -5 2>&1`

## Arguments
$ARGUMENTS

## Task

Create a new stacked branch with the current changes.

**Critical:** `stackit create` requires staged changes. It creates a branch AND commits in one atomic operation.

1. If no staged changes, run `git add --all` first
2. If no changes at all (staged or unstaged), inform user and stop
3. **If changes span multiple unrelated concerns** (different features, mixed refactoring + features, many directories with different purposes), use `AskUserQuestion`:
   - Header: "Large changes"
   - Question: "These changes span multiple areas. How would you like to proceed?"
   - Options:
     - "Use /stack-plan (Recommended)" → Stop and tell user to run `/stack-plan`
     - "Single commit" → Proceed with one commit
     - "Let me describe" → Wait for user to provide message
4. If user provided `-m "message"`, use that message
5. Otherwise, generate a commit message matching the project's style (see recent commits)
6. Run: `echo "<message>" | command stackit create [branch-name] [--scope <scope> if provided] --no-interactive`
7. **If create fails due to missing scope** (error mentions scope required):
   - Use `AskUserQuestion`:
     - Header: "Scope"
     - Question: "Branch pattern requires a scope (e.g., feature area, ticket ID). What scope should this branch use?"
     - Options: Generate 2-3 sensible suggestions based on the codebase + user can type custom
   - Retry with `--scope <value>`

You can call multiple tools in a single response. Stage and create in one message.

**Never use:** `git commit` or `git checkout -b` — always use `stackit create`.

## Follow-up

After successful creation, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Branch created. What would you like to do next?"
- Options:
  - "Submit as PR (Recommended)" → Invoke `/stack-submit` using Skill tool
  - "Stack another change" → Tell user to make changes and run `/stack-create`
  - "Done for now" → End with summary
