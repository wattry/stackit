---
icon: material/account-group
title: Team Collaboration with Stackit
description: Collaborate on stacks with your team. Share configuration, protect branches with freeze/lock, fetch teammate stacks, and coordinate handoffs.
---

# Team Collaboration

Stackit supports team collaboration with shared configuration, branch protection, and stack sharing features.

## Sharing Configuration

### Team Settings (`.stackit.yaml`)

Create a `.stackit.yaml` file in your repository root for team-wide defaults:

```yaml
# .stackit.yaml
trunk: main

# Branch naming
branch:
  pattern: "{username}/{date}/{message}"

# PR settings
submit:
  footer: true
merge:
  method: squash

# CI validation
ci:
  command: "make test"

# PR navigation
navigation:
  when: multiple
  location: body
```

Commit this file to share settings across all contributors.

### Personal Overrides

Individual developers can override team settings locally:

```bash
# Override merge method for yourself
stackit config set merge.method rebase

# Override branch pattern
stackit config set branch.pattern "{message}"
```

Personal settings are stored in `.git/config` and not shared.

### Configuration Hierarchy

Settings are applied in order (highest priority first):

1. **Personal** — `.git/config` (local only)
2. **Team** — `.stackit.yaml` (committed, shared)
3. **Defaults** — Built-in stackit defaults

## Branch Protection

Stackit provides two levels of branch protection with different scopes.

### Freeze (Local Only)

Freezing prevents local modifications to a branch on your machine only:

```bash
# Freeze a branch and its downstack
stackit freeze feature-api

# Unfreeze when you need to make changes
stackit unfreeze feature-api
```

**Freeze prevents:**

- Modifying commits ($$stackit modify$$, $$stackit squash$$)
- Absorbing changes ($$stackit absorb$$)
- Restacking

**Use freeze when:**

- Building on top of a teammate's PR
- Reviewing someone else's stack
- Preventing accidental modifications to shared code

!!! tip
    Frozen branches are automatically updated from remote during $$stackit sync$$ and $$stackit get$$ using hard-resets.

### Lock (Shared)

Locking prevents modifications for everyone collaborating on the stack:

```bash
# Lock a branch (stored in remote metadata)
stackit lock feature-api

# Unlock when ready for changes
stackit unlock feature-api
```

**Lock prevents** the same operations as freeze, but for all collaborators.

**Use lock when:**

- Signaling that a branch is stable and ready for review
- Preventing teammates from modifying branches during review
- Coordinating handoffs between team members

!!! note
    Lock status is stored in remote metadata and shared when others run $$stackit get$$ or $$stackit sync$$.

### Freeze vs Lock

| Aspect | Freeze | Lock |
|--------|--------|------|
| Scope | Local machine only | All collaborators |
| Storage | Local git config | Remote metadata |
| Use case | Building on others' PRs | Coordinating team changes |
| Sync behavior | Hard-reset from remote | Preserves lock status |

## Getting Stacks from Teammates

### Fetching a Stack

Sync a teammate's stack to your machine:

```bash
# By PR number
stackit get 123

# By branch name
stackit get feature-api
```

This:

1. Fetches the branch and all parent branches to trunk
2. Reconstructs the stack structure locally
3. Freezes the fetched branches by default (prevents accidental modifications)

### Checkout Options

By default, fetched branches are frozen. To allow modifications:

```bash
# Fetch without freezing
stackit get 123 --unfrozen

# Or unfreeze later
stackit unfreeze feature-api
```

### Building on a Fetched Stack

After fetching, create your own branches on top:

```bash
# Checkout the top of their stack
stackit checkout their-feature

# Create your branch
git add your-changes.go
stackit create your-feature -m "feat: extend the feature"
```

Your branches are yours to modify; theirs remain frozen.

### Syncing Existing Stacks

If you already have the stack locally, $$stackit get$$ syncs with remote:

```bash
stackit get feature-api
```

Use `--force` to overwrite local changes with the remote version:

```bash
stackit get feature-api --force
```

## CI Integration

### Local CI Validation

Configure a CI command to run across your stack:

```yaml
# .stackit.yaml
ci:
  command: "make test"
  timeout: 600  # seconds
```

Run it with $$stackit foreach$$:

```bash
# Run CI command on each branch
stackit foreach
```

Or run any command:

```bash
# Run tests on all upstack branches
stackit foreach "go test ./..."

# Run on downstack branches instead
stackit foreach --downstack "npm test"
```

### CI Command Configuration

Set the CI command via config:

```bash
stackit config set ci.command "make test"
stackit config set ci.timeout 300
```

## Collaboration Patterns

### Handing Off a Stack

When passing a stack to a teammate:

1. **Lock the branches** you're handing off:

    ```bash
    stackit lock feature-api
    ```

2. **Push the latest changes**:

    ```bash
    stackit submit
    ```

3. **Share the PR link** — your teammate can run:

    ```bash
    stackit get <pr-url>
    ```

### Continuing Someone's Work

When picking up a teammate's stack:

1. **Fetch the stack**:

    ```bash
    stackit get their-branch
    ```

2. **Unfreeze if you need to modify**:

    ```bash
    stackit unfreeze their-branch
    ```

3. **Create your branches on top**:

    ```bash
    stackit create your-continuation -m "feat: continue the work"
    ```

### Pair Programming on Stacks

For active collaboration on the same stack:

1. One person creates the initial stack
2. Both run $$stackit get$$ to sync
3. Use $$stackit lock$$ to coordinate who can modify which branches
4. Regularly sync with $$stackit get$$ to pull each other's changes

### Merging a Teammate's Stack

If you have permission to merge someone else's PRs:

```bash
# Fetch their stack
stackit get their-feature

# Review and merge
stackit merge
```

## Next Steps

- [Daily Workflows →](daily.md)
- [Advanced Workflows →](advanced.md)
- [Configuration →](../cli/config.md)
