---
description: Create a new stacked branch with intelligent naming
allowed-tools: Bash(stackit:*), Bash(git:*), Read
argument-hint: [branch-name] [-m "commit message"]
---

# Stack Create

Create a new stacked branch on top of the current branch.

## Context
- Current branch: !`git branch --show-current`
- Staged changes: !`git diff --cached --stat 2>/dev/null || echo 'No staged changes'`
- Stack state: !`stackit log --oneline 2>/dev/null || echo 'Not initialized'`

## Arguments
$ARGUMENTS

## Instructions

1. Check if there are staged changes with `git diff --cached --stat`
2. If no staged changes, ask user what to stage or use `--all` flag

3. **If branch name provided in arguments**: use it directly
   **If no branch name provided**: generate one:
   - Analyze staged files to understand the change
   - Create kebab-case name (max 50 chars)
   - Examples: "add-user-auth", "fix-login-bug", "refactor-api-client"

4. **For commit message**:
   - **If provided via -m flag**: use it directly
   - **If can be piped**: Accept from stdin (e.g., `echo "message" | stackit create branch-name`)
   - **If generating**:
     - Check for CONTRIBUTING.md in repo root for commit message guidelines
     - Follow project's commit format conventions if documented
     - Otherwise use conventional commit format: type(scope): description
     - Examples: "feat(auth): add user authentication", "fix(api): handle timeout errors"

5. Run: `stackit create <name> -m "<message>"` or `echo "<message>" | stackit create <name>`

6. Show new stack state with `stackit log`

## Error Handling
- If on trunk: warn user they should create from a feature branch or confirm
- If uncommitted changes exist: suggest staging first
- If create fails: show error and suggest fixes
