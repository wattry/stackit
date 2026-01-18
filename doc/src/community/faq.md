---
icon: material/frequently-asked-questions
---

# Frequently Asked Questions

Common questions about stackit.

## General

### What is stacking?

Stacked changes (or "stacked diffs") is a development workflow where you break a large feature into small, focused branches that build on top of each other. Instead of one massive Pull Request, you have a "stack" of smaller PRs.

See [Core Concepts](../guide/concepts.md) for more details.

### Do I need GitHub?

Yes, stackit currently requires GitHub for PR operations. It uses the GitHub CLI (`gh`) to create and manage pull requests.

### Can I use stackit with GitLab or Bitbucket?

Not currently. Stackit is designed specifically for GitHub workflows. Support for other platforms may be added in the future.

## Usage

### Can I use stackit with existing branches?

Yes! Use $$stackit track$$ to start tracking an existing branch:

```bash
git checkout existing-branch
stackit track
```

### What happens if I use regular git commands?

Stackit tracks branches using Git metadata, so most git commands are safe. However:

- Creating branches with `git checkout -b` means they won't be tracked automatically (use $$stackit create$$ instead)
- Force-pushing with `git push --force` can confuse stackit's metadata (use $$stackit submit$$ instead)

When in doubt, use stackit commands for branch operations.

### How do I delete a stack?

To delete a branch and its metadata:

```bash
stackit delete <branch-name>
```

To delete the current branch:

```bash
stackit delete
```

This also updates child branches to point to the deleted branch's parent.

### Can I rename a branch?

Yes, use $$stackit rename$$:

```bash
stackit rename new-branch-name
```

This updates all metadata, PRs, and child branch relationships.

## Advanced

### How does metadata work?

Stackit stores metadata in Git refs under `refs/stackit/metadata/`. This includes:

- Parent-child relationships
- PR information
- Branch scopes
- Lock/freeze status

You don't need to manage this directly—stackit handles it automatically.

### Can I see the raw metadata?

Yes, for debugging purposes:

```bash
# View all stackit refs
git for-each-ref refs/stackit/

# View metadata for a specific branch
git show-ref refs/stackit/metadata/<branch-name>
```

### What happens during $$stackit sync$$?

The sync command:

1. Fetches the latest from remote
2. Pulls the trunk branch
3. Identifies and deletes merged branches
4. Removes associated worktrees for merged stacks
5. Restacks remaining branches on the updated trunk

### How do I undo a command?

Most stackit commands save a snapshot before making changes:

```bash
stackit undo
```

This restores branches and metadata to the state before the last command.

!!! warning
    Undo only works for the most recent command. It's not a full history of operations.

### Can multiple people work on the same stack?

Yes! Use $$stackit get$$ to fetch someone else's stack:

```bash
stackit get <pr-number>
```

By default, fetched branches are frozen to prevent accidental modifications. Use $$stackit unfreeze$$ if you need to modify them.

For shared stacks, use $$stackit lock$$ to signal that branches should not be modified by anyone.

### What's the difference between frozen and locked?

- **Frozen**: Local protection only. Prevents you from modifying branches on your machine.
- **Locked**: Shared protection. Visible to everyone who fetches the stack.

Both prevent operations like modify, squash, and absorb.

## Troubleshooting

### Why am I getting "branch not tracked"?

The branch wasn't created with $$stackit create$$. Fix it by tracking manually:

```bash
stackit track
```

### My PRs have the wrong base branch

Run $$stackit submit$$ to update all PRs with correct base branches:

```bash
stackit submit --stack
```

### How do I recover from a failed restack?

If conflicts occur during restack:

1. Resolve conflicts and stage changes
2. Run $$stackit continue$$

Or abandon the restack:

```bash
stackit abort
```

## More help

- [Troubleshooting guide](../guide/troubleshooting.md)
- [GitHub Issues](https://github.com/getstackit/stackit/issues)
- [CLI Reference](../cli/reference.md)
