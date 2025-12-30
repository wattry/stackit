# Navigation Commands

Commands for moving through your stack and viewing stack state.

> **CRITICAL:** Always run these commands with `--no-interactive`.

## Core Navigation

| Command | Description |
|---------|-------------|
| `stackit log --no-interactive` | Display the branch tree visualization |
| `stackit log --no-interactive full` | Show tree with GitHub PR status and CI checks |
| `stackit checkout --no-interactive` | Interactive branch switcher |

## Movement Commands

| Command | Description |
|---------|-------------|
| `stackit up --no-interactive` | Move to the child branch |
| `stackit down --no-interactive` | Move to the parent branch |
| `stackit top --no-interactive` | Move to the top of the stack |
| `stackit bottom --no-interactive` | Move to the bottom of the stack |
| `stackit trunk --no-interactive` | Return to the main/trunk branch |

## Information Commands

| Command | Description |
|---------|-------------|
| `stackit children` | Show children of current branch |
| `stackit parent` | Show parent of current branch |
| `stackit info --no-interactive` | Show detailed branch info |

## Quick Tips

- Always run `stackit log --no-interactive` first to understand your position
- Use `stackit checkout --no-interactive` for interactive selection when you have many branches
- `stackit log --no-interactive full` shows PR status - useful before submitting
