---
description: View current stack state and health
model: haiku
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Status

Show the current stack state, identify issues, and provide actionable recommendations.
## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`
- Branch info: !`command stackit info --json --no-interactive 2>&1`
- Health report: !`command stackit health --json --no-interactive 2>&1`

## Task

### Step 1: Display Stack Overview

Based on the stack state context, show:
- Current position in the stack (marked with "current")
- Parent/child relationships
- Any branches that need attention

### Step 2: Health Analysis

Parse the health report JSON and highlight any issues:

**High Priority Issues (act now):**
- CI failing on any branch
- Merge conflicts detected

**Medium Priority Issues (address soon):**
- Branches needing restack
- Branches significantly behind trunk (>3 days)

**Low Priority (when convenient):**
- Branches ready to merge (approved + CI passing)
- Branches without PRs

### Step 3: Recommendations

Based on the health report, provide actionable next steps:

| Issue | Recommendation |
|-------|----------------|
| Branches need restack | `command stackit restack --no-interactive` |
| Branches have no PR | `command stackit submit --no-interactive` |
| CI failing | Check CI logs and fix issues |
| Branch ready to merge | `command stackit merge <branch> --no-interactive` |
| Behind trunk | `command stackit sync --no-interactive` |

### Step 4: Summary

End with a brief summary:
- Total branches in stack
- Overall health status (healthy/needs attention/issues found)
- Primary recommended action (if any)

## Example Output

```
Stack Status
============

Current: feature-api (3 commits, +45/-12)
  ↑ feature-models (2 commits, +30/-5) - needs restack
  ↑ main

Health:
  ⚠️  feature-models: needs restack (2 days behind)
  ✓  feature-api: CI passing, ready for review

Recommendations:
  1. Run `command stackit restack` to update feature-models
  2. Run `command stackit submit` to create PR for feature-api

Summary: 2 branches, 1 needs attention
```
