---
description: Submit branches as PRs with auto-generated descriptions
allowed-tools: Bash(stackit:*), Bash(git:*), Read
argument-hint: [--stack | --draft]
---

# Stack Submit

Submit branches to GitHub and create/update PRs.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log --no-interactive 2>&1`
- Branch info: !`stackit info --json --no-interactive 2>&1`

## Arguments
$ARGUMENTS

## Instructions

1. Check the branch info JSON - if parent branch doesn't exist, run `stackit sync` first
2. For branches without PRs, prepare PR metadata:
   - Check for .github/pull_request_template.md for format requirements
   - Title: first commit message line
   - Body: summary of commits + test plan
3. Run submit command:
   - Current branch: `stackit submit --no-interactive`
   - Entire stack: `stackit submit --stack --no-interactive`
   - As drafts: add `--draft`
4. Report created/updated PR URLs

## Do NOT
- Submit if parent branch was deleted (run sync first)
- Create PRs with placeholder or empty descriptions
