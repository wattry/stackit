---
description: Create a new stacked branch with intelligent naming
allowed-tools: Bash(stackit:*), Bash(git:*), Read, AskUserQuestion, Skill
argument-hint: [-m "message"] [branch-name]
---

# Stack Create

Create a new stacked branch on top of the current branch.

## CRITICAL: Required Workflow

**`command stackit create` requires staged changes.** It creates a branch AND commits staged changes in one atomic operation.

```bash
# CORRECT workflow:
git add -A                                    # 1. Stage changes FIRST
echo "message" | command stackit create --no-interactive  # 2. Then create

# WRONG - NEVER do this:
command stackit create -m "..."   # Creates empty branch!
git commit -m "..."       # BYPASSES stackit - FORBIDDEN!
```

## FORBIDDEN Commands

When creating new stacked branches, NEVER use:
- `git commit` - Always use `command stackit create` instead
- `git checkout -b` - Use `command stackit create` which handles both

## Context (pre-staging)
- Current branch: !`git branch --show-current`
- Unstaged changes: !`git diff --stat 2>&1 | head -20`
- Staged changes: !`git diff --cached --stat 2>&1 | head -20`
- Recent commits (for style): !`git log --oneline -5 2>&1`

## Arguments
$ARGUMENTS

## Instructions

### Step 1: Ensure Changes Are Staged

**If staged changes already exist:** Skip to Step 2.

**If only unstaged changes exist:** Auto-stage all changes:
```bash
git add --all
```

**If no changes at all:** Stop and inform the user there's nothing to commit.

### Step 2: Gather Diff Context

After staging is confirmed, get the diff for commit message generation:
```bash
git diff --cached
```

### Step 3: Get Commit Message

**If user provided a message** (via $ARGUMENTS like `-m "message"`):
- Use the provided message directly, skip to Step 4

**Otherwise, generate the commit message inline:**

Using the diff from Step 2 and the recent commits style from Context, generate a commit message:
- Under 72 chars for first line
- Follow Conventional Commits if the project uses them (check recent commits)
- Imperative mood ("Add" not "Added")
- Match the style of recent commits

For simple/obvious changes (single-purpose diffs, documentation, bug fixes), generate directly.

**For large, diverse changes** (many files across different directories, multiple unrelated features/concerns, or significant refactoring spanning multiple systems), use `AskUserQuestion`:
- Header: "Large changes"
- Question: "These changes span multiple areas and may benefit from being split into separate stacked branches. How would you like to proceed?"
- Options:
  - "Use /stack-plan" - Analyze and split into multiple focused branches (Recommended)
  - "Single commit" - Combine all changes into one commit
  - "Let me describe" - I'll provide the commit message

If user selects "Use /stack-plan", inform them: "Run `/stack-plan` to analyze your changes and create a well-organized stack." Then stop - do not create a branch.

For moderately complex changes (large diff but single concern), use `AskUserQuestion`:
- Header: "Commit scope"
- Question: "This is a large diff. How should I structure the commit?"
- Options:
  - "Single commit" - Combine all changes with a summary message
  - "Let me describe" - I'll provide the commit message

### Step 4: Create the Branch

Run the create command. Use a heredoc for messages with special characters:

```bash
echo "<commit_message>" | command stackit create [branch-name] --no-interactive
```

Or with heredoc for complex messages:
```bash
command stackit create [branch-name] --no-interactive <<'EOF'
<commit_message>
EOF
```

- Branch name is optional; stackit auto-generates from commit message

The command outputs the result including the new branch name. No additional commands needed.

## Do NOT
- Run `command stackit create` without staged changes (creates empty branch)
- Use `git commit` after creating a branch - this bypasses stackit
- Run `git status` or `git diff --stat` redundantly - trust the context
- Run `stackit log` after create - the create output is sufficient

## Follow-up

After successful branch creation, use `AskUserQuestion`:
- Header: "Next step"
- Question: "Branch created successfully. What would you like to do next?"
- Options:
  - label: "Submit as PR (Recommended)"
    description: "Push branch and create/update pull request"
  - label: "Stack another change"
    description: "Create another branch on top of this one"
  - label: "Done for now"
    description: "No follow-up action needed"

Based on response:
- **"Submit as PR"**: Invoke `/stack-submit` skill using the `Skill` tool
- **"Stack another change"**: Tell user: "Make your changes, stage them with `git add -A`, then run `/stack-create`"
- **"Done for now"**: End with summary: "Branch `<name>` created. Run `/stack-submit` when ready to create a PR."
