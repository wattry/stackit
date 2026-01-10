---
description: Intelligently fold granular branches into their parents
allowed-tools: Bash(stackit:*), Bash(git:*), Read
---

# Stack Fold

Fold (squash) granular branches into their parent branches.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log --no-interactive 2>&1`
- Stack info: !`stackit info --stack --json --no-interactive 2>&1`

## Instructions

1. Analyze the stack info JSON to identify fold candidates:
   - Small branches (few commits, small diffs)
   - Minor fix messages ("fix typo", "address review", "tweak")
   - Must not be locked or frozen
   - Parent must not be locked or frozen
   - Must match parent's scope
2. Propose a fold plan to user with reasoning
3. Run `stackit fold --dry-run --no-interactive` to preview
4. Before each fold, verify:
   - Branch has no children (fold leaf branches first)
   - Parent still not locked/frozen
5. Execute: `stackit checkout <branch> && stackit fold --no-interactive`
6. Run `stackit restack --no-interactive` after folding
7. Show final stack state

## Do NOT
- Fold into trunk unless user explicitly requests it
- Fold locked or frozen branches
- Fold across different scopes
- Fold branches that have children (fold children first)
