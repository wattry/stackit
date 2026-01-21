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

Run `/stackit` for the full guide.
