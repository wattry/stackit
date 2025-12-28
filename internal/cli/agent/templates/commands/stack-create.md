---
description: Create a new stacked branch with intelligent naming
allowed-tools: Bash(stackit:*), Bash(git:*), Read
argument-hint: [optional-branch-name]
---

# Stack Create

Create a new stacked branch on top of the current branch. Branch name is optional - stackit will generate one from the commit message.

## Context
- Current branch: !`git branch --show-current`
- Staged changes: !`git diff --cached --stat 2>/dev/null || echo 'No staged changes'`
- Stack state: !`stackit log --oneline 2>/dev/null || echo 'Not initialized'`

## Arguments
$ARGUMENTS

## Instructions

1. Check if there are staged changes with `git diff --cached --stat`
2. If no staged changes, ask user what to stage or use `--all` flag

3. **Generate commit message**:
   - Check for CONTRIBUTING.md or README.md for commit message guidelines
   - Follow project's commit format conventions if documented
   - Otherwise use conventional commit format: type(scope): description
   - Examples: "feat(auth): add user authentication", "fix(api): handle timeout errors"

4. **Branch name** (optional):
   - If user provided branch name in arguments: use it
   - Otherwise: stackit will auto-generate from commit message

5. **Run command using pipe** (preferred):
   - With branch name: `echo "commit message" | stackit create branch-name`
   - Auto-generate name: `echo "commit message" | stackit create`

6. Show new stack state with `stackit log`

## Error Handling
- If on trunk: warn user they should create from a feature branch or confirm
- If uncommitted changes exist: suggest staging first
- If create fails: show error and suggest fixes
