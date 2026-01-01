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
- `-m "message"` - Commit message
- `--all` - Stage all changes first
- `--insert` - Insert between current and child

**Preferred usage:** `echo "commit message" | stackit create [optional-name]`

### stackit modify
- Similar to `git commit --amend` but updates stack metadata
- Auto-restacks children if needed

### stackit absorb
- Automatically determines which commit each change belongs to
- **Warning:** May cause compilation errors if dependencies split across commits
- Always validate with build/test after absorb
