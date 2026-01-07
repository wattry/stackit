# Branch Management Commands

Commands for creating, modifying, and managing individual branches.

> **CRITICAL:** Always run these commands with `--no-interactive`.

## Creating and Modifying

| Command | Description |
|---------|-------------|
| `stackit create [name]` | Create a new stacked branch (name optional, auto-generated) |
| `stackit modify` | Amend current commit (like git commit --amend) |
| `stackit absorb` | Auto-amend changes to correct commits in stack |

## Branch Lifecycle

| Command | Description |
|---------|-------------|
| `stackit split` | Split current branch into multiple branches |
| `stackit squash` | Squash all commits on current branch |
| `stackit fold` | Merge current branch into its parent |
| `stackit pop` | Delete branch but keep changes in working tree |
| `stackit delete` | Delete current branch and metadata |
| `stackit rename [name]` | Rename current branch |

## Metadata

| Command | Description |
|---------|-------------|
| `stackit scope [name]` | Manage logical scope (Jira/Linear ID) |
| `stackit track` | Start tracking a branch |
| `stackit untrack` | Stop tracking a branch |

## Common Flag Patterns

### stackit create
- `--all` - Stage all changes first
- `--insert` - Insert between current and child
- `-w` - Create with a dedicated worktree

**Preferred usage (pipe format):**
```bash
# Branch name auto-generated from commit message
echo "feat: add user authentication" | stackit create --no-interactive

# With explicit branch name
echo "feat: add user authentication" | stackit create my-branch --no-interactive

# Stage all and create
echo "feat: add user authentication" | stackit create --all --no-interactive
```

### Multiple commits per branch
**A stacked branch can have multiple commits.** You don't need a new branch for every commit:
- Add another commit: `git add . && git commit -m "message"`
- Amend current commit: `stackit modify --no-interactive`
- Absorb fixes to correct commits: `stackit absorb --no-interactive`

### stackit modify
- Similar to `git commit --amend` but updates stack metadata
- Auto-restacks children if needed

### stackit absorb
- Automatically determines which commit each change belongs to
- **Warning:** May cause compilation errors if dependencies split across commits
- Always validate with build/test after absorb
