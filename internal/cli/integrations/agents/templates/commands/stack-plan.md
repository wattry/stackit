---
description: Plan and create a stack from uncommitted working tree changes
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Glob, Grep, AskUserQuestion
argument-hint: [check-command]
---

# Stack Plan

Analyze uncommitted working tree changes, suggest a logical stacking structure, and create the stack with build verification at each step.

**Primary objective: Never lose someone's work.**

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- Stack state: !`command stackit log --no-interactive 2>&1`

## Arguments
$ARGUMENTS

## Instructions

### Phase 1: Gather All Changes

First, check the current state of the working tree:

```bash
# Check for existing stashes (warn if present)
git stash list

# Check current staging state
git diff --cached --stat

# Staged changes
git diff --cached

# Unstaged changes
git diff

# List untracked files
git ls-files --others --exclude-standard
```

**If existing stashes found**: Warn the user:
```
Warning: You have existing stashes. Stack-plan will create additional stash
entries during execution. Your existing stashes will not be affected, but
recovery instructions will reference the most recent stash created by stack-plan.
```

For untracked files, read their contents to understand what they contain.

**For binary files**: Note the filename and extension. If the purpose is unclear, use `AskUserQuestion`:
- Header: "Binary file"
- Question: "What is the purpose of <filename>? Which feature does it belong to?"
- Options: List candidate branches or "Other"

**If no changes detected**: Inform the user and exit - there's nothing to stack.

**Note on partial staging**: If changes are partially staged, all changes will be re-staged during execution. The original staging state is not preserved.

### Phase 2: Analyze and Propose Stack Structure

Analyze the changes and determine a logical stacking order using these strategies:

**Grouping Strategies** (pick best fit for the codebase):
1. **By concern/feature** - Group related functionality together
2. **By layer** - Group by architectural layer (db → api → frontend)
3. **By dependency order** - Foundational changes first, dependent changes later
4. **By directory** - When directories cleanly map to concerns

**Ordering Principles**:
- Independent/foundational changes come first (bottom of stack)
- Changes that depend on others come later (top of stack)
- Smaller, focused branches are preferred over large ones

**Limitation - File-level granularity**: This workflow splits changes at the file level. If a single file contains changes that should go to different branches (e.g., two unrelated functions modified), ask the user to either:
1. Accept the file going to one branch, OR
2. Manually split the file's changes first using `git add -p`, then re-run /stack-plan

**Present the proposal clearly**:
```
Proposed Stack Structure
========================
1. [branch-name-1] "commit message 1"
   - file1.go
   - file2.go
   Reason: <why these are grouped and why they're first>

2. [branch-name-2] "commit message 2"
   - file3.go
   - tests/file3_test.go
   Reason: <why these are grouped and depend on branch 1>

3. [branch-name-3] "commit message 3"
   - file4.go
   Reason: <why this comes last>
```

**Validate branch names** before proposing:
```bash
# Check if branch names already exist locally
git branch --list <branch-name>

# Check if branch names exist on remote
git ls-remote --heads origin <branch-name> 2>/dev/null
```

If a proposed branch name already exists, choose a different name or ask the user.

**If a file logically belongs in multiple branches**, use `AskUserQuestion`:
- Header: "File placement"
- Question: "Which branch should contain <filename>?"
- Options: List the candidate branch names

**If unsure about grouping**, use `AskUserQuestion`:
- Header: "Grouping"
- Question: "How should I group these related files?"
- Options: Present the reasonable grouping strategies

### Phase 3: Determine Build/Check Command

**If provided in arguments**: Use that command.

**Otherwise, auto-detect**:
1. Check for `justfile` → use `just check` or `just test`
2. Check for `Makefile` → use `make test` or `make check`
3. Check for `package.json` → use `npm test` or `npm run check`
4. Check README.md or CONTRIBUTING.md for build/test instructions

**If no command found**, use `AskUserQuestion`:
- Header: "Check command"
- Question: "What command should I use to verify each branch builds correctly?"
- Options:
  - "Skip verification" - Create branches without running checks (not recommended)
  - "Let me specify" - User will provide the command

If user selects "Let me specify", wait for them to provide the command in chat.

### Phase 4: Pre-validate Working Set

Before creating any branches, verify the combined changes build successfully:

```bash
# 1. Stage ALL changes
git add -A

# 2. Run the build/check command
<check-command>
```

**Note**: Pre-validation confirms the combined changes build together. Individual branches are still verified separately during execution, as splitting might expose missing dependencies between files.

**On pre-validation success**: Report success and continue to approval:
```
Pre-validation Passed
=====================
Your combined changes pass the build. Ready to split into branches.
```

**On pre-validation failure**: Report the error:
```
Pre-validation Failed
=====================
Your changes don't pass the build. Fix the issues first before stacking.

Error output:
<error-details>

Your changes are intact - nothing has been committed or branched.
```

Then use `AskUserQuestion` to determine next steps:
- Header: "Pre-validation failed"
- Question: "The build failed before creating any branches. How would you like to proceed?"
- Options:
  - "I'll fix it" - Exit so user can fix issues and re-run /stack-plan
  - "Skip validation" - Proceed without pre-validation (not recommended)
  - "Cancel" - Exit without creating branches

**Why pre-validate?**
- Catches issues before any branches are created
- If the combined changes don't build, splitting them won't help
- User can fix issues with all changes still in working tree (easiest state to work with)

### Phase 5: Get User Approval

Present the complete plan with validation status:

```
Stack Plan Summary
==================
Branches to create: N
Total files: Y
Pre-validation: PASSED ✓ (or SKIPPED if user chose to skip)

1. [branch-name-1] "commit message 1" (X files)
2. [branch-name-2] "commit message 2" (Y files)
3. [branch-name-3] "commit message 3" (Z files)

Each branch will be verified with: <check-command>
```

Use `AskUserQuestion` to get approval:
- Header: "Stack plan"
- Question: "Ready to create this stack?"
- Options:
  - "Execute" - Create all branches with verification
  - "Dry run" - Show exact commands without executing
  - "Modify" - Change groupings, order, or names
  - "Cancel" - Exit without creating branches

**If user selects "Dry run"**, show the exact commands that would run for each branch:
```
Dry Run - Commands for Branch 1 of 3: <branch-name>
===================================================
git reset HEAD
git add file1.go file2.go
git stash push --include-untracked -m "stack-plan-<timestamp>: remaining changes for branches 2-3"
echo "commit message" | command stackit create <branch-name> --no-interactive
<check-command>
git stash apply
git stash drop

[Continue for each branch...]
```

Then ask again whether to execute or cancel.

**If user selects "Modify"**, let them describe changes, update the plan, and ask for approval again.

### Phase 6: Execute Stack Creation

Before starting, display the recovery reference:

```
Starting Stack Creation
=======================
Total branches to create: N
Progress: [0/N]

Recovery Reference:
- STATE A: All changes uncommitted → Nothing to recover
- STATE B: Some branches created, rest stashed → Run: git stash apply && git stash drop
- STATE D: Build failed mid-execution → Stash auto-restored, see options

If anything fails, your changes are recoverable. Proceeding...
```

**For each branch**, follow these steps:

#### Step 1: Report Current State

For **branch 1** (first branch):
```
[1/N] Creating branch: <branch-name>
=====================================
Current state: STATE A (all changes uncommitted)
Files for this branch: <file-list>
Files remaining after this: <remaining-file-list>
```

For **branches 2 through N-1** (middle branches):
```
[X/N] Creating branch: <branch-name>
=====================================
Current state: STATE B
- Branches created: [branch-1, branch-2, ...]
- Stash contains: <remaining-files> (for branches X+1 to N)
- Files for this branch: <file-list>

If something goes wrong:
1. Run: git stash apply && git stash drop
2. Your remaining changes will be restored to working tree
3. Created branches are preserved
```

For **branch N** (last branch):
```
[N/N] Creating branch: <branch-name> (final)
============================================
Current state: STATE B
- Branches created: [branch-1, ..., branch-N-1]
- Files for this branch: <file-list>
- No stash needed (this is the last branch)
```

#### Step 2: Execute Branch Creation

**For branches 1 through N-1** (not the last branch):
```bash
# 1. Reset staging (clean slate)
git reset HEAD

# 2. Stage only files for this branch
git add <file1> <file2> ...

# 3. Stash remaining changes with timestamp for identification
git stash push --include-untracked -m "stack-plan-$(date +%s): remaining changes for branches X+1 to N"

# 4. Create the stacked branch
echo "<commit-message>" | command stackit create <branch-name> --no-interactive

# 5. Run build verification
<check-command>

# 6. Restore stashed changes (apply then drop for safety)
git stash apply
git stash drop
```

**For branch N** (last branch - no stash needed):
```bash
# 1. Reset staging (clean slate)
git reset HEAD

# 2. Stage the remaining files
git add <file1> <file2> ...

# 3. Create the stacked branch (no stash - these are the last files)
echo "<commit-message>" | command stackit create <branch-name> --no-interactive

# 4. Run build verification
<check-command>
```

**Why stash?** Without stashing, files destined for later branches remain in the working tree. Tests could pass falsely by seeing code that won't be committed to this branch.

**Why `git stash apply` + `git stash drop` instead of `git stash pop`?** `stash pop` can fail with conflicts and leave you in an ambiguous state. `stash apply` is safer - if it fails, the stash is still intact for manual recovery.

#### Step 3: Handle Results

**On build success**: Update progress and continue to the next branch:
```
[X/N] ✓ Branch <branch-name> created and verified
```

**On build failure**: STOP immediately. First restore stash (if not last branch):
```bash
git stash apply
git stash drop
```

Then report:
```
[X/N] ✗ Build Failed
====================
Branch: <branch-name>
Error output:
<error-details>

Current state: STATE D
- Branches created: [branch-1, ..., branch-X]
- Remaining changes: Restored to working tree

Your remaining changes are safe.
```

Then use `AskUserQuestion`:
- Header: "Build failed"
- Question: "The build failed on branch <branch-name>. How would you like to proceed?"
- Options:
  - "Continue" - I've fixed the issue, resume from next branch
  - "Undo last" - Rollback <branch-name> with `command stackit undo`
  - "Rollback all" - Undo all created branches and restore original state
  - "Stop here" - Keep created branches, remaining changes stay in working tree

**If user selects "Rollback all"**:
```bash
# Undo each created branch in reverse order
command stackit undo  # Undo branch X
command stackit undo  # Undo branch X-1
# ... repeat for all created branches
```
Then report: "All branches rolled back. Your original changes are in the working tree."

**On stash apply failure** (conflicts):
```
Stash Recovery Failed
=====================
git stash apply failed, likely due to conflicts.

Your stash is still intact. To recover manually:
1. git stash list                    # Find the stack-plan stash
2. git stash show stash@{0}          # Verify it's the right one
3. git checkout --theirs .           # If you want to discard conflicts
4. git stash apply stash@{0}         # Try again
5. git stash drop stash@{0}          # Clean up after success
```

### Phase 7: Report Completion

After all branches are created successfully:

```bash
command stackit log --no-interactive
```

Present a summary:
```
Stack Created Successfully
==========================
[N/N] Complete ✓

Created N branches from Y files:
1. ✓ branch-name-1 (X files)
2. ✓ branch-name-2 (Y files)
3. ✓ branch-name-3 (Z files)

Stack structure:
<stackit log output>

Next steps:
- Run /stack-submit to create PRs
- Run /stack-verify to re-verify all branches
```

## Error Handling Reference

| Phase | State | Scenario | Recovery |
|-------|-------|----------|----------|
| Gather | A | No changes detected | Inform user and exit |
| Gather | A | Existing stashes present | Warn user, continue |
| Analyze | A | Branch name already exists | Choose different name or ask user |
| Analyze | A | File needs hunk-level split | Ask user to split manually or accept file placement |
| Pre-validate | A | Build fails | User fixes with all changes intact, or skips validation |
| Approval | A | User cancels | Exit - changes intact in working tree |
| Approval | A | User requests dry run | Show commands, ask again |
| Execution 1 | A | Stash fails | Check for conflicts, report, ask user |
| Execution 1 | A | Create fails | No stash to restore, report error |
| Execution 2+ | B | Stash fails | Previous branches intact, report, ask user |
| Execution 2+ | B | Create fails | `git stash apply && git stash drop`, report |
| Execution 2+ | B | Build fails | `git stash apply && git stash drop`, report STATE D, offer options |
| Execution 2+ | B | Stash apply fails | Stash intact, provide manual recovery steps |
| Execution N | B | Build fails on last branch | No stash to restore, report, offer undo options |
| Complete | C | Success | All changes committed across stack |

**Key guarantees**:
- Pre-validation failure: All changes remain in working tree (STATE A)
- Mid-execution failure: Stash is always restored before stopping (STATE D)
- Stash apply used instead of pop: On failure, stash remains intact for manual recovery
- No scenario should leave changes inaccessible

## Do NOT
- Create branches without user approval of the plan
- Continue past a build failure
- Put the same file in multiple branches
- Use `git commit` directly - always use `command stackit create`
- Make up branch names without showing the user first
- Skip build verification (unless user explicitly says to)
- Use `git stash pop` - always use `git stash apply` + `git stash drop`
- Propose branch names that already exist
