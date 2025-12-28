# Navigation Commands

Commands for moving through your stack and viewing stack state.

## Core Navigation

| Command | Description |
|---------|-------------|
| `stackit log` | Display the branch tree visualization |
| `stackit log full` | Show tree with GitHub PR status and CI checks |
| `stackit checkout` | Interactive branch switcher |

## Movement Commands

| Command | Description |
|---------|-------------|
| `stackit up` | Move to the child branch |
| `stackit down` | Move to the parent branch |
| `stackit top` | Move to the top of the stack |
| `stackit bottom` | Move to the bottom of the stack |
| `stackit trunk` | Return to the main/trunk branch |

## Information Commands

| Command | Description |
|---------|-------------|
| `stackit children` | Show children of current branch |
| `stackit parent` | Show parent of current branch |
| `stackit info` | Show detailed branch info |

## Quick Tips

- Always run `stackit log` first to understand your position
- Use `stackit checkout` for interactive selection when you have many branches
- `stackit log full` shows PR status - useful before submitting
