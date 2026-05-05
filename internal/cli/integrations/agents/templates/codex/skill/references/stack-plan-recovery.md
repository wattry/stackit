# Stack Plan Recovery

`stack-plan` creates a backup branch before splitting changes. The backup is deleted only after all planned branches are created and verified.

## Partial Run State

A partial run may leave:

- A branch named `stack-plan-backup-<timestamp>` with all original changes.
- Some created stacked branches.
- The original branch with an empty or partial working tree.

## Start Over

```bash
git checkout stack-plan-backup-<timestamp>
git checkout -B <original-branch>
git reset --hard stack-plan-backup-<timestamp>
```

Then delete partial stacked branches newest-first with:

```bash
stackit undo --no-interactive --yes
```

## Continue From The Last Successful Branch

```bash
stackit log --no-interactive
git checkout stack-plan-backup-<timestamp>
git diff <last-created-branch>..stack-plan-backup-<timestamp> --stat
git checkout <last-created-branch>
git checkout stack-plan-backup-<timestamp> -- <files-for-next-branch>
printf '%s\n' "<commit message>" | stackit create -F - <next-branch> --no-interactive
```

## Clean Up

Only after confirming every intended change is in the stack:

```bash
git branch -D stack-plan-backup-<timestamp>
```
