---
description: Create a new stacked branch with intelligent naming
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Task
argument-hint: [optional-branch-name]
---

# Stack Create

Create a new stacked branch on top of the current branch.

## CRITICAL: Required Workflow

**You MUST stage changes BEFORE running `command stackit create`.** The command creates a branch AND commits staged changes in one atomic operation.

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

## Context
- Current branch: !`git branch --show-current`
- Staged changes: !`git diff --cached --stat`
- Staged diff (first 200 lines): !`git diff --cached | head -200`
- Recent commits: !`git log --oneline -5 2>&1`
- Stack state: !`command stackit log --no-interactive 2>&1`

**Note:** The staged diff is truncated to 200 lines. For very large changes, focus on the diff stats and file names to understand scope, then read specific files if needed for context.

## Arguments
$ARGUMENTS

## Instructions

### Step 1: Check for Staged Changes

If no staged changes exist:
- Ask user what to stage, OR
- Run `git add --all` if user confirms

### Step 2: Check Current Position

If on trunk (main/master):
- Warn user and confirm they want to create a stack from trunk

### Step 3: Get Commit Message

**If user provided a message** (via $ARGUMENTS like `-m "message"`):
- Use the provided message directly, skip to Step 4

**Otherwise, generate using subagent:**

Use the Task tool to spawn a **haiku** subagent. The context from the inline commands above (staged diff, recent commits) is already available - use those values directly in the subagent prompt.

Gather additional context only for project conventions:
```bash
# Check for commit conventions in project docs
head -100 CONTRIBUTING.md 2>/dev/null || head -50 README.md 2>/dev/null || echo ""
```

**Spawn the commit message subagent:**

Use Task tool with `model: haiku` and `subagent_type: general-purpose`. Insert the actual values from the Context section above:

```
Generate a commit message for these staged changes.

Diff stats:
<INSERT: actual output from "Staged changes" context>

Diff content:
<INSERT: actual output from "Staged diff" context>

Recent commits (for style reference):
<INSERT: actual output from "Recent commits" context>

Project conventions (if any):
<INSERT: conventions from CONTRIBUTING.md/README.md, or "None documented">

Generate a commit message:
- Under 72 chars for first line
- Match style of recent commits
- Follow project conventions if documented
- Imperative mood ("Add" not "Added")

Respond with:
MESSAGE:
<the commit message>
END_MESSAGE
```

**Parse the subagent response:** Extract the commit message between `MESSAGE:` and `END_MESSAGE` markers.

### Step 4: Create the Branch

Run the create command with the generated message. Use a heredoc to handle multi-line messages and special characters safely:

```bash
command stackit create [branch-name] --no-interactive <<'EOF'
<commit_message>
EOF
```

Or for single-line messages without special characters:
```bash
echo "<commit_message>" | command stackit create [branch-name] --no-interactive
```

- Branch name is optional; stackit auto-generates from commit message
- Use heredoc (`<<'EOF'`) if the message contains quotes, newlines, or special shell characters

### Step 5: Confirm Success

Show the new stack state:

```bash
command stackit log --no-interactive
```

## Do NOT
- Run `command stackit create` without staged changes (creates empty branch)
- Use `git commit` after creating a branch - this bypasses stackit
- Generate commit messages with the main model when no message was provided (use haiku subagent)
