---
icon: material/puzzle
title: Stackit Integrations
description: Integrate stackit with AI agents, GitHub Actions, Git hooks, and shell for a seamless stacking workflow.
---

# Integrations

Stackit integrates with your development tools to provide a seamless stacking workflow.

## Available Integrations

<div class="grid cards" markdown>

-   :material-robot:{ .lg .middle } **AI Agents**

    ---

    AI-assisted stacking with intelligent commands and skill files for Claude Code, Codex, and Cursor.

    [Setup agents →](claude.md)

-   :material-github:{ .lg .middle } **GitHub**

    ---

    GitHub Actions for CI checks on locked branches and stack order enforcement.

    [Setup GitHub →](github.md)

-   :material-hook:{ .lg .middle } **Git Hooks**

    ---

    Pre-commit and pre-push hooks to prevent modifications to locked branches.

    [Setup hooks →](git-hooks.md)

-   :material-console:{ .lg .middle } **Shell Integration**

    ---

    Enable automatic directory changes when working with worktrees.

    [Setup shell →](shell.md)

</div>

## Quick Setup

Install all recommended integrations:

```bash
# Agent integration (Claude Code + Codex + Cursor)
stackit agent install

# GitHub Actions
stackit github install

# Git hooks (pre-commit and pre-push)
stackit precommit install
stackit prepush install

# Shell integration (add to your shell config)
eval "$(stackit shell zsh)"  # or bash/fish
```

## Integration Overview

| Integration | Purpose | Setup Command |
|:------------|:--------|:--------------|
| AI Agents (Claude/Codex/Cursor) | AI-assisted stacking commands | `stackit agent install` |
| GitHub | CI checks for PRs | `stackit github install` |
| Pre-commit hook | Block commits to locked branches | `stackit precommit install` |
| Pre-push hook | Block pushes to locked branches | `stackit prepush install` |
| Shell | Auto-cd for worktrees | `eval "$(stackit shell zsh)"` |
