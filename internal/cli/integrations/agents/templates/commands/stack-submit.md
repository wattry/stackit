---
description: Submit branches as PRs with auto-generated descriptions
model: sonnet
allowed-tools: Bash(stackit:*), Bash(git:*), Read, AskUserQuestion, Skill
argument-hint: [--stack | --draft]
---

# Stack Submit

Submit branches to GitHub and create/update PRs.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`command stackit log --no-interactive 2>&1`
- Branch info: !`command stackit info --json --no-interactive 2>&1`

## Arguments
$ARGUMENTS

## Instructions

### Step 1: Pre-flight Checks

Check the branch info JSON:
- If parent branch doesn't exist or was deleted, run `command stackit sync --no-interactive` first
- If there are uncommitted changes, warn the user

### Step 2: Identify Branches Needing PRs

From the stack state, identify which branches need PR descriptions generated:
- Branches without existing PRs need full title + body generation
- Branches with PRs may just need updates

### Step 3: Submit PRs

Run submit command:

**Current branch only:**
```bash
command stackit submit --no-interactive
```

**Entire stack:**
```bash
command stackit submit --stack --no-interactive
```

**As drafts:** add `--draft`

When stackit prompts for title and body (in interactive mode) or when generating PR content:
- **Title**: Use first commit subject, keep under 72 chars
- **Body**: Summarize changes with bullet points + include test plan

Check `.github/pull_request_template.md` and `CONTRIBUTING.md` for format requirements.

### Step 4: Report Results

Report the created/updated PR URLs to the user.

## Do NOT
- Submit if parent branch was deleted (run sync first)
- Create PRs with placeholder content ("TODO", "TBD", empty sections)

## Follow-up

After successful submit, use `AskUserQuestion`:
- Header: "Next step"
- Question: "PRs submitted successfully. What would you like to do next?"
- Options:
  - label: "Sync with trunk (Recommended)"
    description: "Pull latest changes from trunk and update stack"
  - label: "Check PR status"
    description: "View full stack status with PR links"
  - label: "Done for now"
    description: "No follow-up action needed"

Based on response:
- **"Sync with trunk"**: Invoke `/stack-sync` skill using the `Skill` tool
- **"Check PR status"**: Run `command stackit log full --no-interactive`
- **"Done for now"**: End with summary including PR URLs from the submit output
