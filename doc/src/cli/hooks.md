---
icon: material/hook
---

# Git Hooks

Stackit provides Git hooks to enforce branch restrictions locally, preventing commits or pushes to locked or frozen branches.

## Pre-commit hook

The pre-commit hook prevents commits to locked or frozen branches. This catches issues early, before you've written a commit message.

### Installation

```bash
stackit precommit install
```

This creates (or appends to) `.git/hooks/pre-commit` with:

```bash
#!/bin/bash
# Installed by Stackit. To bypass, use --no-verify.
stackit precommit verify
```

### Behavior

When you run `git commit` on a locked or frozen branch:

```bash
$ git commit -m "my changes"
Error: branch my-feature is locked (user)
```

### Uninstall

```bash
stackit precommit uninstall
```

## Pre-push hook

The pre-push hook prevents pushing locked or frozen branches to the remote. This is useful when collaborating on stacks, as it prevents accidentally pushing changes to branches that should remain stable.

### Installation

```bash
stackit prepush install
```

This creates (or appends to) `.git/hooks/pre-push` with:

```bash
#!/bin/bash
# Installed by Stackit. To bypass, use --no-verify.
stackit prepush verify
```

### Behavior

When you try to push a locked branch:

```bash
$ git push origin my-locked-branch
Error: cannot push branch "my-locked-branch": branch my-locked-branch is locked (user)
```

The hook checks all branches being pushed. Untracked branches (not managed by stackit) are always allowed.

### Uninstall

```bash
stackit prepush uninstall
```

## Bypassing hooks

Both hooks can be bypassed when necessary using Git's `--no-verify` flag:

```bash
git commit --no-verify -m "emergency fix"
git push --no-verify origin my-branch
```

!!! warning
    Use `--no-verify` sparingly. The hooks exist to prevent accidental modifications to locked branches.

## Installing during init

When you run $$stackit init$$, you're offered the option to install the pre-commit hook. You can also install hooks at any time using the commands above.

## How locking works

Branches can be locked for different reasons:

| Lock Reason | Description |
|:---|:---|
| `user` | Manually locked with $$stackit lock$$ |
| `consolidating` | Temporarily locked during multi-branch operations |

A locked branch cannot be modified until it's unlocked with $$stackit unlock$$.

### Frozen branches

Branches fetched with $$stackit get$$ are **frozen** by default. Frozen branches can be modified locally but serve as a signal that you're working on someone else's code. Unfreeze with $$stackit unfreeze$$.

## Commands reference

| Command | Description |
|:---|:---|
| `stackit precommit install` | Install pre-commit hook |
| `stackit precommit uninstall` | Remove pre-commit hook |
| `stackit precommit verify` | Check if current branch can be modified |
| `stackit prepush install` | Install pre-push hook |
| `stackit prepush uninstall` | Remove pre-push hook |
| `stackit prepush verify` | Check if branches being pushed can be modified |

## Next steps

- [Learn about branch locking →](../guide/concepts.md#frozen-vs-locked-branches)
- [Configuration options →](config.md)
