---
description: Extract commits or files to an independent branch
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Glob, Grep
---

# Stack Extract

Extract commits or file changes from the current branch to a new independent branch.

## Context
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -5`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Instructions

### Use Case 1: Extract file changes to sibling branch (Recommended)

If the user wants to extract specific file(s) to an independent branch:

```bash
command stackit split --by-file <file-path> --as-sibling --name "<branch-name>"
```

This creates a new branch on the same parent (usually main) with just those file changes.
The current branch is unchanged - files are NOT removed.

**Example:**
```bash
# Extract docs changes to a sibling branch
command stackit split --by-file docs/README.md --as-sibling --name "docs-update"
```

### Use Case 2: Extract to parent branch (Default split behavior)

If the extracted files should logically come before the current changes (dependency extraction):

```bash
command stackit split --by-file <file-path>
```

This creates the split branch as a NEW PARENT of the current branch.
The files ARE removed from the current branch.

### Use Case 3: Extract specific commits (Manual workflow)

If you need to extract specific commits (not files):

1. Identify the commit SHA(s) to extract:
   ```bash
   git log --oneline -10
   ```

2. Create new branch from the target parent:
   ```bash
   git checkout main
   git checkout -b <new-branch-name>
   git cherry-pick <commit-sha>
   command stackit track
   ```

3. Return to original branch and remove the commit:
   ```bash
   git checkout <original-branch>
   git reset --hard HEAD~N  # N = number of commits extracted
   command stackit restack --no-interactive
   ```

### Verification

After extraction, verify:
```bash
command stackit log  # Both branches should be siblings on the same parent
just check           # All checks should pass
```

## Choosing Between Sibling and Parent

| Use --as-sibling when... | Use default (parent) when... |
|--------------------------|------------------------------|
| Changes don't belong in this stack | Changes are a dependency for this stack |
| You want an independent PR | The split should be merged first |
| Files should exist on both branches | Files should only be on the split branch |

## Do NOT
- Use raw git commands when stackit commands are available
- Forget to verify the stack structure after extraction
- Extract all files from a branch (at least one must remain)
