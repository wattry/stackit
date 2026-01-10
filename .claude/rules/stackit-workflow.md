# Stackit Workflow Rules

## MANDATORY: Use Stackit for All Branch/Commit Operations

This project uses stackit for stacked changes. You MUST use stackit commands, NEVER raw git commands.

## Forbidden Commands

NEVER use these commands:
- `git commit` - Use `stackit create` instead
- `git checkout -b` - Use `stackit create` instead
- `gh pr create` - Use `stackit submit` instead
- `git rebase` - Use `stackit restack` instead

## Required Workflow

When creating a new stacked change:

1. **Make code changes first**
2. **Stage changes**: `git add -A`
3. **Create stacked branch with commit**: `stackit create -m "type: description"`

```bash
# CORRECT workflow:
git add -A
stackit create -m "feat: add new feature"

# WRONG - never do this:
stackit create -m "feat: add new feature"  # Creates empty branch!
git commit -m "feat: add new feature"       # Bypasses stackit!
```

## Why This Matters

- `stackit create` expects staged changes and creates the branch + commit together
- Running `stackit create` without staged changes creates an empty branch
- Using `git commit` after that bypasses stackit's metadata tracking
- This breaks the stack structure and PR relationships

## Skills to Use

Prefer using skills over manual commands:
- `/stack-create` - Handles the workflow correctly
- `/stack-submit` - Submit PRs for the stack
- `/stack-status` - Check stack health before operations
