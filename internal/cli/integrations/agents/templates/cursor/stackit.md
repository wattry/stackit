# Stackit Agent Rules

This repository uses [stackit](https://github.com/getstackit/stackit) for managing stacked branches.

## Overview

Stackit is a CLI tool for managing stacked changes in Git repositories. Stacks can be linear chains or branch into tree structures when you need parallel work paths. When working with this codebase, you should use stackit commands to create, manage, and submit branches.

**CRITICAL:** Always use the `--no-interactive` flag when running any `stackit` command. For commands that require confirmation, also include the `--force` flag (for absorb) or `--yes` flag (for undo/merge).

## Key Commands

### Creating Branches

- `stackit create [name] --no-interactive` - Create a new branch stacked on top of the current branch
  - If no name is provided, branch name is generated from commit message
  - Use `-m "message"` to specify a commit message
  - Use `--all` to stage all changes before creating
  - Use `--insert` to insert a branch between current and its child

### Managing Stacks

- `stackit log --no-interactive` - Display the branch tree visualization
- `stackit up --no-interactive` - Move up one branch in the stack
- `stackit down --no-interactive` - Move down one branch in the stack
- `stackit top --no-interactive` - Move to the top of the stack
- `stackit bottom --no-interactive` - Move to the bottom of the stack

### Submitting PRs

- `stackit submit --no-interactive` - Submit the current branch and its ancestors as PRs
- `stackit submit --stack --no-interactive` - Submit the entire stack (including descendants)
- Use `--draft` to create draft PRs
- Use `--edit` to interactively edit PR metadata

### Other Useful Commands

- `stackit sync --no-interactive` - Sync branches with remote
- `stackit restack --no-interactive` - Restack branches to fix conflicts
- `stackit fold --no-interactive` - Fold a branch into its parent
- `stackit split --no-interactive` - Split a branch into multiple branches

## Workflow Guidelines

1. **When creating a new feature**: Use `stackit create --no-interactive` to create a new branch on top of the current branch
2. **When making changes**: Commit normally, stackit tracks the relationships
3. **When ready to submit**: Use `stackit submit --no-interactive` to create/update PRs for the stack
4. **When conflicts occur**: Use `stackit restack --no-interactive` to resolve and restack branches

## Important Notes

- Always use stackit commands with `--no-interactive` instead of raw git commands for branch management
- Stackit maintains parent-child relationships between branches (a branch can have multiple children)
- PRs are automatically created/updated with proper base branches
- The stack structure (linear or tree-shaped) is preserved when submitting multiple PRs

For more information, see: https://github.com/getstackit/stackit
