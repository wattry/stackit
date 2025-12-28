---
name: stackit
description: Manage stacked Git branches with Stackit. Use this skill when creating branches, submitting PRs, navigating stacks, or troubleshooting stack issues.
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Grep, Glob
version: 1.0.0
---

# Stackit - Stacked Branch Management

You are an expert at using Stackit to manage stacked Git branches. Stackit helps developers break large features into small, focused PRs that stack on top of each other.

## Before Any Operation

Always run `stackit log` first to understand:
- Current branch position in the stack
- Parent/child relationships
- Which branches need attention

Check for project conventions:
- Read CONTRIBUTING.md if present for project-specific guidelines
- Follow documented commit message formats, PR templates, and workflows
- Respect any branching or testing requirements specified

## Core Workflows

### Creating a New Branch
1. Stage changes: `git add <files>` or use `--all` flag
2. Generate commit message following project conventions
3. Create branch (name is optional, auto-generated from message):
   - Preferred: `echo "commit message" | stackit create`
   - With name: `echo "commit message" | stackit create branch-name`

### Submitting PRs
1. Check stack state: `stackit log`
2. Submit current + ancestors: `stackit submit`
3. Submit entire stack: `stackit submit --stack`
4. For drafts: `stackit submit --draft`

### Syncing with Main
1. Run `stackit sync` to pull trunk and cleanup merged branches
2. If branches were deleted, run `stackit restack`

### Fixing Issues
1. For rebase conflicts: resolve files, then `stackit continue`
2. To abort: `stackit abort`
3. To undo: `stackit undo`

### Fixing Compilation Errors After Absorb
After `stackit absorb`, compilation errors may occur when absorbed changes depend on files/changes that didn't get cleanly absorbed:

1. Check README.md and CONTRIBUTING.md for build/test commands
2. For each branch in stack (bottom to top):
   - Run build/test commands
   - If failures: analyze errors for missing dependencies
   - Check upstack branches for needed changes: `git diff <branch>..<child>`
   - Apply missing changes (cherry-pick or manual copy)
   - Verify fix by re-running build/test
3. Use `stackit foreach "<command>"` to verify entire stack builds

## Auto-Generation Guidelines

### Branch Names
Branch names are optional - stackit auto-generates from commit message:
- Only provide if user explicitly requests a specific name
- Auto-generated format: kebab-case from commit message
- Example: "Add user authentication" -> "add-user-authentication"

### Commit Messages
When generating commit messages:
- Check README.md and CONTRIBUTING.md for project-specific guidelines
- Follow documented commit message conventions if available
- Default to conventional commit format: type(scope): description
- **Always use pipe format**: `echo "message" | stackit create`

### PR Descriptions
When generating PR descriptions:
- Check for .github/pull_request_template.md or CONTRIBUTING.md for PR format
- Follow project-specific PR templates if available
- Default format:
  - Title: First commit message line
  - Body: Bullet points from all commits in branch
  - Include "## Test Plan" section with testing steps
  - Add any required sections from project templates

## Important Rules

1. **Never use raw git for branch operations** - always use stackit commands
2. **Check state before destructive operations** - run `stackit log` first
3. **Handle conflicts gracefully** - guide user through resolution
4. **Keep PRs small and focused** - suggest splitting if too large
