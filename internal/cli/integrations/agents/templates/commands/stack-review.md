---
description: Apply actionable PR review comments and mark them resolved
allowed-tools: Bash(gh:*), Bash(stackit:*), Bash(git:*), Read, Edit, Grep, Glob, Task
argument-hint: [--dry-run]
---

# Stack Review

Fetch PR review comments for the current branch, evaluate them, apply actionable changes, and mark resolved.

## Context
- Current branch: !`git branch --show-current`
- Repo: !`gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null`
- PR info: !`gh pr view --json number,url,state,reviewDecision,headRefName 2>/dev/null || echo "No PR for this branch"`

## Arguments
$ARGUMENTS

## Instructions

### Step 1: Verify PR Exists

If there's no PR for the current branch, inform the user and stop.

### Step 2: Parse Repository Information

Extract owner and repo from the `nameWithOwner` context value (format: `owner/repo`):

```bash
# Example: if nameWithOwner is "myorg/myrepo"
# owner = "myorg"
# repo = "myrepo"
```

Extract the PR number from the PR info JSON's `number` field.

### Step 3: Fetch Review Threads

Get all review threads with their comments and resolution status:

```bash
gh api graphql -f query='
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          originalLine
          originalStartLine
          diffSide
          comments(first: 20) {
            nodes {
              id
              body
              author { login }
              createdAt
            }
          }
        }
      }
    }
  }
}' -f owner=<OWNER> -f repo=<REPO> -F pr=<NUMBER>
```

Replace `<OWNER>`, `<REPO>`, and `<NUMBER>` with the parsed values from Step 2.

**Pagination note:** This query fetches up to 100 threads with 20 comments each. For PRs with more threads, you would need to paginate using `after` cursor, but this covers the vast majority of PRs.

### Step 4: Filter Threads

Focus only on threads that are:
- **Not resolved** (`isResolved: false`)
- **Not outdated** (`isOutdated: false`) - outdated means the code has changed
- **Have a file path** (`path` is not null) - skip PR-level comments without file context

Also filter out bot comments by checking `author.login` for common bot patterns:
- Names ending in `[bot]` (e.g., `dependabot[bot]`, `github-actions[bot]`)
- Known CI bots: `codecov`, `sonarcloud`, `renovate`

Skip threads that are already resolved, outdated, or only contain bot comments.

### Step 5: Batch Read Files

**Optimization:** Before evaluating threads, collect all unique file paths and read them upfront:

```bash
# Get unique paths from filtered threads
# Read each file once and cache for later use
```

This avoids reading the same file multiple times when multiple threads reference it.

### Step 6: Evaluate Each Thread

For each unresolved thread, examine the code from the cached file content.

**Handle line number edge cases:**
- If `line` is `null`: This is a file-level comment (applies to whole file, not a specific line)
- If `startLine` differs from `line`: This is a multi-line comment spanning `startLine` to `line`
- Use `diffSide` to determine if comment is on old (`LEFT`) or new (`RIGHT`) code
- **Line drift:** If `line` differs from `originalLine`, the code may have shifted. Use the comment body and `originalLine` context to locate the actual code being referenced. Search for distinctive patterns mentioned in the comment.

**For many threads (5+):** Use a **haiku** subagent with the `review-triage` template to batch-classify threads efficiently. The subagent returns JSON with actionable/not-actionable classifications.

**Classify as ACTIONABLE if the comment:**
- Requests a specific code change ("rename X to Y", "add null check", "remove this")
- Points out a bug with a clear fix ("this will NPE", "missing return")
- Requests error handling, validation, or safety improvements
- Asks for documentation/comments on specific code

**Classify as NOT ACTIONABLE if the comment:**
- Is a question without a clear answer ("why is this here?")
- Is a discussion or debate without resolution
- Is praise or approval ("LGTM", "nice work")
- Requires clarification from the author before acting
- Is vague or ambiguous about what change is needed
- Is a file-level comment without specific change request

**Be conservative:** When in doubt, skip the comment rather than make an incorrect change.

### Step 7: Apply Changes (or Dry Run)

**If `--dry-run` is specified:**
- List each actionable comment and what change would be made
- Do NOT modify any files
- Do NOT resolve any threads
- Stop here

**Otherwise, for each actionable comment:**

1. Read the relevant file
2. Locate the code using `path` and `line` from the thread
3. Apply the suggested change using the Edit tool
4. **Verify the edit succeeded** - if the Edit tool reports an error, mark this thread as failed
5. Track successful changes for the commit message

### Step 8: Commit Changes

If any changes were successfully applied, stage and commit them.

**Note:** Using `git commit` here is acceptable because we're adding to an existing stacked branch, not creating a new one. The `stackit create` command is for creating new branches with commits.

```bash
git add -A
git commit -m "refactor: apply PR review feedback

Applied reviewer suggestions:
- <bullet list of successful changes>"
```

### Step 9: Mark Threads as Resolved

**Only resolve threads where:**
- The edit was successfully applied (verified in Step 7)
- The commit succeeded (verified in Step 8)

For efficiency, batch multiple thread resolutions into a single GraphQL mutation using aliases:

```bash
gh api graphql -f query='
mutation {
  t1: resolveReviewThread(input: {threadId: "THREAD_ID_1"}) {
    thread { isResolved }
  }
  t2: resolveReviewThread(input: {threadId: "THREAD_ID_2"}) {
    thread { isResolved }
  }
}'
```

Replace `THREAD_ID_1`, `THREAD_ID_2`, etc. with the actual thread IDs. Use unique aliases (`t1`, `t2`, etc.) for each mutation.

**Do NOT resolve threads where:**
- The edit failed or was skipped
- The commit failed
- The change couldn't be verified

### Step 10: Restack

If changes were committed, restack to update dependent branches:

```bash
command stackit restack --no-interactive
```

If restack fails with conflicts:
1. Check `git status` for conflicted files
2. Read each conflicted file
3. Resolve conflicts keeping the review feedback changes where appropriate
4. Continue with `command stackit continue --no-interactive`

### Step 11: Report Results

Provide a summary:

```
## Review Summary

**Threads processed:** N
**Changes applied:** N
**Threads resolved:** N
**Threads skipped:** N
**Threads failed:** N

### Applied Changes
- file.go:42 - Added error handling per @reviewer
- utils.go:15 - Renamed variable for clarity

### Skipped Threads
- file.go:88 - Question about design (needs discussion)
- main.go:12 - File-level comment without specific change

### Failed Threads (if any)
- config.go:23 - Edit failed: could not locate code block
```

## Do NOT
- Apply changes from vague or unclear comments - skip them
- Guess at what a reviewer meant - if unclear, skip
- Modify files when `--dry-run` is specified
- Resolve threads that weren't actually addressed or where edits failed
- Apply changes that would break compilation without fixing
- Force push or modify history that's already been reviewed
