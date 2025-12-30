---
description: Intelligently absorb working changes into correct commits
allowed-tools: Bash(stackit:*), Bash(git:*), Bash(~/.claude/skills/stackit/scripts/*:*), Read, Grep
---

# Stack Absorb

Absorb working directory changes into the correct commits in your stack using intelligent heuristics.

## Context
- Current branch: !`git branch --show-current`
- Changed files: !`git status --short 2>/dev/null`
- Stack structure: !`stackit log --no-interactive 2>/dev/null`

## What Absorb Does

`stackit absorb --no-interactive --force` automatically assigns each change in your working directory to the "most appropriate" commit in your stack based on:
- Which commit last modified each line
- File proximity and context
- Change patterns

This is powerful but can occasionally cause compilation errors when dependencies split across commits.

## Instructions

1. **Show user what will be absorbed:**
   ```bash
   git status
   git diff
   ```

2. **Explain absorb risks clearly:**
   - Changes go to "best guess" commits (usually correct, but not always)
   - May cause compilation errors if dependencies split across commits
   - Example: Function definition → commit A, function usage → commit B
   - Result: Commit B fails to build (function not defined yet)
   - **This is normal and fixable** - see fix-absorb workflow

3. **Confirm user wants to proceed** (absorb modifies git history)

4. **Run absorb:**
   ```bash
   stackit absorb --no-interactive --force
   ```

5. **CRITICAL: Validate stack after absorb**

   This is **NOT OPTIONAL**. Absorb can cause compilation errors.

   **Validation loop:**
   ```bash
   # Check project for build/test commands
   grep -i "build\|test" README.md CONTRIBUTING.md

   # Test each branch in stack (bottom to top)
   stackit foreach --no-interactive "<build-command>"

   # If ANY failures:
   # - STOP and mark absorb as incomplete
   # - Use fix-absorb workflow
   # - See ../skills/stackit/workflows/fix-absorb.md
   ```

   **Checklist pattern for validation:**
   ```
   Post-Absorb Validation:
   - [ ] Identified build command
   - [ ] Built each branch in stack
   - [ ] All branches build successfully
   - [ ] No compilation errors
   ```

6. **If ALL builds pass:**
   - Show final stack state: `stackit log --no-interactive`
   - Confirm absorb completed successfully

7. **If ANY builds fail:**
   - **IMMEDIATELY switch to fix-absorb workflow**
   - See [../skills/stackit/workflows/fix-absorb.md](../skills/stackit/workflows/fix-absorb.md)
   - Use the detailed checklist in that workflow
   - Do NOT consider absorb complete until all branches build

## Success Criteria

Absorb is only complete when:
- ✓ All changes absorbed from working directory
- ✓ All branches in stack build without errors
- ✓ All tests pass (if applicable)
- ✓ Stack structure is clean

## Example Walkthrough

```bash
# 1. Check what will be absorbed
git status
# → 3 files modified

# 2. Run absorb
   stackit absorb --no-interactive --force
# → Changes absorbed into commits

# 3. CRITICAL: Validate
just build  # or npm build, cargo build, etc.
# → Build failed!

# 4. Don't panic - use fix-absorb workflow
# See ../skills/stackit/workflows/fix-absorb.md
# → Follow checklist to find and fix dependencies

# 5. After fix, verify entire stack
stackit foreach --no-interactive "just build"
# → All branches build successfully

# 6. Now absorb is complete
stackit log --no-interactive
# → Clean stack structure
```

## Common Patterns

### Pattern 1: All Builds Pass

This is the happy path - absorb worked perfectly:
```bash
   stackit absorb --no-interactive --force
stackit foreach --no-interactive "just build"
# → All passed ✓
# Done!
```

### Pattern 2: Some Builds Fail

This is common and expected:
```bash
   stackit absorb --no-interactive --force
stackit foreach --no-interactive "just build"
# → Branch 'add-validation' failed
# → Branch 'add-api' failed

# Use fix-absorb workflow
# See ../skills/stackit/workflows/fix-absorb.md
# Fix dependencies, then verify again
```

### Pattern 3: Abort If Too Complex

If absorb created too many issues:
```bash
stackit undo --no-interactive --yes  # Restore to pre-absorb state
# Manually create commits instead
```

## Error Handling

- **If absorb command fails:** Show error, suggest committing changes manually
- **If builds break:** MUST use fix-absorb workflow (not optional)
- **If conflicts occur:** Use `stackit abort --no-interactive` to recover, then see absorb-conflict workflow
- **Never leave stack in broken state** - always complete validation loop

## Conflict Resolution

If absorb encounters a merge conflict:

1. **Diagnose the conflict:**
   ```bash
   stackit absorb --show-conflict --no-interactive
   ```
   This shows staged changes, stack structure, and guidance.

2. **Abort and recover:**
   ```bash
   stackit abort --no-interactive
   ```
   This returns you to your original branch and restores any stashed changes.

3. **Resolution strategies:**
   - **Split the change:** Use `git add -p` to stage only parts that absorb cleanly
   - **Create new commit:** Use `stackit create --no-interactive` for changes that don't belong in existing commits
   - **Manual application:** Checkout target branch and apply changes directly

See [../skills/stackit/workflows/absorb-conflict.md](../skills/stackit/workflows/absorb-conflict.md) for detailed conflict resolution guidance.

## Prevention Tips

For better absorb results:
1. **Keep changes focused** - absorb works best with related changes
2. **Absorb frequently** - smaller change sets = fewer issues
3. **Use modify for small fixes** - `stackit modify` is safer for targeted changes
4. **Validate immediately** - catch issues early

## Comparison with Modify

| Operation | Use When | Risks |
|-----------|----------|-------|
| `stackit modify` | Amending current commit only | Low - changes go to known commit |
| `stackit absorb --no-interactive --force` | Multiple commits need updates | Medium - heuristic assignment |

## Important Notes

- Absorb modifies git history (like rebase)
- Always validate with build/test after absorb
- The fix-absorb workflow is your friend - use it
- When in doubt, use `stackit modify` for targeted changes
