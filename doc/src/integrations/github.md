---
icon: material/github
title: GitHub Actions Integration
description: Automated CI checks for stackit branches. Prevent merging locked PRs and enforce correct merge order with GitHub Actions.
---

# GitHub Integration

Stackit provides GitHub Actions integration to enforce stack discipline in CI/CD pipelines.

## Overview

The GitHub integration adds automated checks to your pull requests:

- **Lock check** — Prevents merging PRs for locked branches
- **Stack order check** — Ensures PRs are merged in the correct order (bottom of stack first)

## Installation

Install the GitHub Action workflow:

```bash
stackit github install
```

This creates `.github/workflows/stackit.yml` in your repository.

!!! note
    You'll need to commit and push this file for the checks to take effect.

### Force Overwrite

If the workflow file already exists:

```bash
stackit github install --force
```

## How It Works

### Lock Check

When a PR is opened or updated, the action checks if the branch is locked:

```
✓ Lock check passed - branch is not locked
```

If the branch is locked:

```
✗ Lock check failed - branch "feature-api" is locked
```

This prevents accidentally merging branches that are still being worked on or coordinated with teammates.

### Stack Order Check

The action verifies that the PR is at the bottom of its stack:

```
✓ Stack order check passed - PR is ready to merge
```

If there are PRs below this one in the stack:

```
✗ Stack order check failed - merge PR #42 first
```

This ensures stacked PRs are merged in order, maintaining clean git history.

## Workflow File

The installed workflow looks like:

```yaml
name: Stackit Checks

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  stackit-checks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install stackit
        run: |
          curl -fsSL https://get.stackit.dev | bash

      - name: Run lock check
        run: stackit github check-lock

      - name: Run stack order check
        run: stackit github check-order
```

## Customization

### Branch Protection Rules

For maximum effectiveness, configure GitHub branch protection:

1. Go to **Settings → Branches → Branch protection rules**
2. Add a rule for your main branch
3. Enable **Require status checks to pass before merging**
4. Select the stackit checks

### Required Checks

You can make either check required or optional:

- **Lock check as required** — Strictly prevents merging locked branches
- **Stack order as advisory** — Shows warning but allows merge (for flexibility)

## Troubleshooting

### Checks Not Running

1. Verify the workflow file exists:
   ```bash
   cat .github/workflows/stackit.yml
   ```

2. Check that it's committed and pushed:
   ```bash
   git status
   ```

3. View workflow runs in GitHub Actions tab

### Check Failing Unexpectedly

1. View the check details in the PR
2. Run the check locally:
   ```bash
   stackit github check-lock
   stackit github check-order
   ```

3. Check branch metadata:
   ```bash
   stackit info
   ```

## Combining with Local Hooks

For complete protection, use both GitHub Actions and local git hooks:

```bash
# GitHub Actions (CI)
stackit github install

# Local hooks (development)
stackit precommit install
stackit prepush install
```

This provides:

- **Local protection** — Catch issues before pushing
- **CI protection** — Enforce rules for all contributors

## Next Steps

- [Git Hooks →](git-hooks.md) — Local pre-commit and pre-push hooks
- [Team Collaboration →](../workflows/collaboration.md) — Branch locking patterns
- [Other integrations →](index.md)
