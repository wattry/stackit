---
description: Submit branches as PRs with auto-generated descriptions
allowed-tools: Bash(stackit:*), Bash(git:*), Read
argument-hint: [--stack | --draft]
---

# Stack Submit

Submit branches to GitHub and create/update PRs.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log --no-interactive 2>/dev/null`

## Arguments
$ARGUMENTS

## Instructions

1. Run `stackit log --no-interactive` to see branches to submit

2. For each branch that will be submitted, gather commit info:
   `git log --oneline <parent>..<branch>`

3. If PRs don't exist yet, prepare PR metadata:
   - Check for .github/pull_request_template.md or CONTRIBUTING.md for PR format requirements
   - **Title**: First commit message line (without conventional commit prefix for cleaner titles)
   - **Body**: Generate from commits following any templates found, or use default:
     ```
     ## Summary
     - <bullet point for each commit>

     ## Test Plan
     - [ ] <suggested test steps based on changes>

     <add any additional sections required by project templates>
     ```

4. **Validate PR metadata quality**:
   - Title is clear and descriptive (not placeholder)?
   - Body has meaningful description (not just TODOs)?
   - Test plan is specific (not empty placeholders)?
   - If validation fails: revise and re-validate
   - Only proceed when PR meets quality standards

5. Run submit command:
   - Current branch only: `stackit submit --no-interactive`
   - Entire stack: `stackit submit --stack --no-interactive`
   - As drafts: `stackit submit --draft --no-interactive`

6. Report created/updated PR URLs

## Error Handling
- If branch needs restack: run `stackit restack --no-interactive` first
- If push fails: check for upstream changes
- If PR creation fails: show GitHub error
