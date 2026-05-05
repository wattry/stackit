---
description: Use when the user wants to create a new Stackit stacked branch from current working tree changes. Trigger phrases include "stack these changes", "create a stacked change", "make a stackit branch", "commit this with stackit", and "turn this into a stacked PR". Stages changes and runs `stackit create -F -`.
---

# Stack Create

Create one new stacked branch from the current working tree.

## Workflow

1. Inspect state:

   ```bash
   git status --short
   git diff --stat
   git diff --cached --stat
   stackit log --no-interactive
   git log --oneline -5
   ```

2. If there are no staged or unstaged changes, tell the user and stop.

3. If the diff obviously spans unrelated review units, use `stack-plan` instead.

4. Stage the intended changes:

   ```bash
   git add -A
   ```

5. Generate a Conventional Commit message matching project style.

6. Create the branch:

   ```bash
   printf '%s\n' "<commit message>" | stackit create -F - --no-interactive
   ```

   With an explicit branch name:

   ```bash
   printf '%s\n' "<commit message>" | stackit create -F - <branch-name> --no-interactive
   ```

   If config requires a scope, add `--scope <value>`.

7. Verify:

   ```bash
   stackit log --no-interactive
   ```

Report the branch name, parent branch, and commit subject.

## Do Not

- Use `git commit` to create the branch.
- Use `git checkout -b`.
- Chain staging and creation in one shell command.
