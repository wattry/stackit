# Stackit Workflow Rules

This project uses stackit for stacked changes. **NEVER use raw git commands for branch/commit operations.**

## Forbidden → Required

| Never Use | Use Instead |
|-----------|-------------|
| `git commit -m "..."` | `command stackit create -m "..."` |
| `git checkout -b` | `command stackit create -m "..."` |
| `gh pr create` | `command stackit submit` |
| `git rebase` | `command stackit restack` |

**Exception:** `git commit` is allowed when adding commits to an existing stacked branch.

## Required Workflow

```bash
# 1. Make changes
# 2. Stage changes FIRST (required!)
git add -A
# 3. Create stacked branch with commit
command stackit create -m "feat: description"
# 4. Submit when ready
command stackit submit
```

**Critical:** `stackit create` requires staged changes. Without staged changes, it creates an empty branch.

## Skills (Preferred)

Use skills instead of manual commands:

| Skill | Purpose |
|-------|---------|
| `/stack-create` | Create stacked branch (handles workflow correctly) |
| `/stack-submit` | Submit PRs for the stack |
| `/stack-status` | Check stack health |
| `/stack-fix` | Diagnose and fix issues |
| `/stack-sync` | Sync with trunk, cleanup merged branches |
| `/stack-restack` | Rebase all branches in stack |
| `/stack-tidy` | Clean up fixup/WIP commits across the stack |

Run `/stackit` for the full guide.

## Common Pitfalls

| Mistake | Fix |
|---------|-----|
| Forgetting to stage before `create` | Always `git add -A` before `stackit create` |
| Empty branch created | You forgot to stage; delete branch and retry with staged changes |
| Using `git commit` for new branch | Use `stackit create` - it creates branch + commit together |
| Using `git checkout -b` | Use `stackit create` - branch name auto-generated from message |
| Manual rebase broke stack | Use `stackit restack` to safely rebase all children |
| Using `gh pr create` | Use `stackit submit` - it handles stacked PR dependencies |
| Amending wrong commit | Use `stackit absorb` to auto-route changes to correct commits |
| Stack out of sync after merge | Run `stackit sync` to cleanup merged branches and update trunk |
