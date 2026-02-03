# Metadata Storage

Stackit stores branch relationships and PR information as Git refs pointing to JSON blobs. This approach keeps metadata portable, versioned, and synchronized with `git push`/`git fetch`.

## Storage Locations

| Ref Pattern | Purpose | Pushed to Remote? |
|-------------|---------|-------------------|
| `refs/stackit/metadata/{branch}` | Branch relationships, PR info | Yes |
| `refs/stackit/local-metadata/{branch}` | Local-only state | No |
| `refs/stackit/remote-metadata/{branch}` | Cached remote metadata | No (fetched copy) |

### How It Works

1. Metadata is serialized as JSON
2. JSON is stored as a Git blob (`git hash-object -w`)
3. A ref points to that blob (`git update-ref`)

This means:
- `git push origin refs/stackit/metadata/*` syncs metadata to remote
- `git fetch origin refs/stackit/metadata/*:refs/stackit/remote-metadata/*` fetches remote metadata
- Metadata is garbage collected like any other Git object

## Main Metadata (`refs/stackit/metadata/`)

This is the primary metadata, pushed to remote and shared across collaborators.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `parentBranchName` | string | Parent branch in the stack |
| `parentBranchRevision` | string | SHA of parent when relationship was created |
| `prInfo` | object | PR number, title, body, state, base, URL, draft status |
| `scope` | string | Branch scope/prefix (e.g., "[PROJ-123]") |
| `lockReason` | enum | Why PR is locked: `""`, `"user"`, `"consolidating"` |
| `branchType` | enum | `"user"`, `"utility"`, `"worktree-anchor"` |
| `lastModifiedBy` | object | Git user who last modified (name, email, GitHub username) |
| `lastModifiedAt` | timestamp | When metadata was last changed |
| `localOnlyHash` | string | Hash of local-only state for change detection |
| `mergedDownstack` | array | Historical parents when branches merged (max 5) |
| `stackDescription` | object | Stack-level title/description (root branch only) |

### PR Info Fields

| Field | Type | Description |
|-------|------|-------------|
| `number` | int | PR number on GitHub |
| `base` | string | Base branch for the PR |
| `url` | string | PR URL |
| `title` | string | PR title |
| `body` | string | PR body/description |
| `state` | string | PR state (OPEN, MERGED, CLOSED) |
| `isDraft` | bool | Whether PR is a draft |
| `lockReason` | string | PR-level lock reason |
| `mergeBranch` | string | Branch to merge into for consolidation |

### Branch Types

| Type | Description |
|------|-------------|
| `user` | Normal stacked branch created by user |
| `utility` | Created by `stackit merge --consolidate` or internal operations |
| `worktree-anchor` | Anchor branch for worktree, has no commits |

### Lock Reasons

| Reason | Description |
|--------|-------------|
| `""` (empty) | Branch is not locked |
| `user` | Manually locked by user (`stackit lock`) |
| `consolidating` | Being consolidated, locked automatically |

## Local Metadata (`refs/stackit/local-metadata/`)

Machine-specific state that should never be pushed to remote.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `frozen` | bool | Whether branch is frozen (excluded from restack) |
| `needsPRBodyUpdate` | bool | Flag for syncing PR body on next submit |
| `navigationCommentId` | int64 | GitHub comment ID for navigation comment |

## Remote Metadata (`refs/stackit/remote-metadata/`)

Cached copy of remote metadata, fetched via:

```bash
git fetch origin 'refs/stackit/metadata/*:refs/stackit/remote-metadata/*'
```

Stackit automatically configures this refspec. Remote metadata is used for:
- Detecting conflicts between local and remote changes
- Syncing metadata from collaborators
- Determining what needs to be pushed

## JSON Examples

### Branch Metadata

```json
{
  "parentBranchName": "main",
  "parentBranchRevision": "abc123def456",
  "prInfo": {
    "number": 42,
    "base": "main",
    "url": "https://github.com/org/repo/pull/42",
    "title": "feat: add user authentication",
    "state": "OPEN",
    "isDraft": false
  },
  "branchType": "user",
  "lastModifiedBy": {
    "gitName": "Alice",
    "gitEmail": "alice@example.com",
    "githubUsername": "alice"
  },
  "lastModifiedAt": "2024-01-15T10:30:00Z"
}
```

### Stack Description (Root Branch Only)

```json
{
  "parentBranchName": "main",
  "stackDescription": {
    "title": "User Authentication Feature",
    "description": "Implements OAuth2 login flow with support for Google and GitHub providers."
  }
}
```

### Local Metadata

```json
{
  "frozen": false,
  "needsPRBodyUpdate": true,
  "navigationCommentId": 123456789
}
```

### Merged Downstack History

When branches are merged or deleted, their history is preserved:

```json
{
  "parentBranchName": "main",
  "mergedDownstack": [
    {
      "branchName": "feat/auth-base",
      "prNumber": 40,
      "prState": "MERGED"
    },
    {
      "branchName": "feat/auth-oauth",
      "prNumber": 41,
      "prState": "MERGED"
    }
  ]
}
```

## Inspecting Metadata

### View metadata for a branch

```bash
# Get the ref SHA
git show-ref refs/stackit/metadata/my-branch

# Read the blob content
git cat-file -p <sha>
```

### List all metadata refs

```bash
git for-each-ref refs/stackit/metadata/
```

### View local metadata

```bash
git for-each-ref refs/stackit/local-metadata/
```

## Implementation

The metadata system is implemented in:
- `internal/git/metadata.go` - Low-level read/write operations
- `internal/engine/` - Higher-level operations through the Engine interface

Always use the Engine interface to query or modify metadata, never the git package directly from application code.
