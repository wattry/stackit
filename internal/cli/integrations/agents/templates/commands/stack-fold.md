---
description: Intelligently fold granular branches into their parents
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Grep
---

# Stack Fold

Analyze the stack to identify "too granular" branches (e.g., small typo fixes, minor adjustments) and recommend folding (squashing) them into their parent branches to maintain stack health and reduce PR noise.

## Context
- Current branch: !`git branch --show-current`
- Stack structure: !`stackit log --no-interactive 2>/dev/null`
- Detailed stack info (JSON): !`stackit info --stack --json --no-interactive 2>/dev/null`

## Identifying Candidates

Analyze the stack to find folding candidates based on these criteria:

1.  **Granularity**: The branch has few commits (often just 1) and small diff stats.
2.  **Descriptiveness**: Commit messages suggest minor fixes or follow-ups (e.g., "fix typo", "address review", "tweak styles").
3.  **Safety**:
    - `is_locked`: Must be `false`.
    - `is_frozen`: Must be `false`.
    - `scope`: Must match the parent's scope (folding across scopes is forbidden).
    - **Target Parent**: The parent branch must also not be locked or frozen.
    - **Trunk**: Never fold into the trunk branch unless explicitly requested.

## Instructions

1.  **Identify candidates**:
    Use `stackit info --stack --json --no-interactive` to analyze branch sizes and metadata.

2.  **Propose a fold plan**:
    Present recommendations to the user, including:
    - **Branch to fold**: The name of the granular branch.
    - **Target parent**: The branch it will be folded into.
    - **Reasoning**: Why it's a good candidate.
    - **Impact**: Show the combined diff or commit messages if possible using:
      ```bash
      stackit checkout <branch-to-fold> --no-interactive
      stackit fold --dry-run --no-interactive
      ```

3.  **Execute the fold**:
    Wait for user confirmation, then for each confirmed branch:
    ```bash
    stackit checkout <branch-to-fold> --no-interactive
    stackit fold --no-interactive
    ```
    *Note: Use `--keep` if the user wants to keep the name of the branch being folded rather than the parent's name.*

4.  **Post-fold cleanup**:
    Ensure the stack remains healthy and synchronized:
    ```bash
    stackit restack --no-interactive
    ```

5.  **Final verification**:
    Show the updated stack state:
    ```bash
    stackit log --no-interactive
    ```

## Safety Constraints

- **Never fold into trunk** unless the user explicitly provides `--allow-trunk`.
- **Never fold locked or frozen branches.**
- **Never fold across different scopes.**
- If you are unsure if a fold is appropriate, ask the user for clarification.
