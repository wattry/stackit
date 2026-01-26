---
icon: material/robot
---

# Claude Code Integration

Stackit includes specialized commands designed for Claude Code, providing intelligent automation for common stacking workflows.

## Overview

Claude Code integration enables AI-assisted stacking with commands that:

- **Understand Context**: Analyze your current stack state and git status
- **Provide Validation**: Include quality checks and error handling
- **Guide Through Issues**: Offer step-by-step resolution guidance
- **Ensure Safety**: Prioritize data safety with undo capabilities

## Setup

Install the Claude integration files:

```bash
stackit agent install
```

This creates the necessary integration files for Claude Code to use specialized stacking commands.

## Available commands

### stack-status

View current stack state, branch position, and health status.

```bash
stack-status
```

**When to use**: Getting oriented in a complex stack, checking for issues.

### stack-create

Create a new stacked branch with intelligent naming and commit messages.

```bash
stack-create [branch-name]
```

**When to use**: Adding a new feature branch to your stack.

### stack-submit

Submit branches as PRs with auto-generated descriptions.

```bash
stack-submit [--stack | --draft]
```

**When to use**: Creating or updating pull requests.

**Options**:
- `--stack`: Submit entire stack
- `--draft`: Submit as draft PRs

### stack-sync

Sync with trunk, cleanup merged branches, and restack.

```bash
stack-sync
```

**When to use**: Keeping your stack up-to-date with main.

### stack-restack

Rebase all branches to ensure proper ancestry.

```bash
stack-restack
```

**When to use**: Fixing branch relationships after changes.

### stack-absorb

Intelligently absorb working changes into correct commits.

```bash
stack-absorb
```

**When to use**: Applying fixes across multiple stack branches, with conflict resolution guidance.

### stack-fix

Diagnose and fix common stack issues.

```bash
stack-fix
```

**When to use**: Resolving compilation errors or structural problems.

## Example workflow

Here's a typical Claude-assisted workflow:

```bash
# Claude helps create a branch with proper commit message
stack-create add-user-auth

# Make your changes...

# Intelligently distribute changes across commits
stack-absorb

# Diagnose and fix any issues
stack-fix

# Create/update all PRs in the stack
stack-submit --stack
```

## How it works

Claude Code commands are executed through skills that:

1. **Analyze current state**: Check git status, stack structure, and branch relationships
2. **Execute operations**: Perform the requested action with appropriate flags
3. **Handle errors**: Provide context-aware error messages and recovery steps
4. **Validate results**: Ensure operations completed successfully

## Integration with regular commands

Claude commands complement regular stackit commands. You can mix and match:

```bash
# Use Claude for complex operations
stack-create my-feature

# Use regular commands for simple tasks
stackit log
stackit checkout parent-branch

# Back to Claude for multi-branch operations
stack-absorb
```

## Best practices

1. **Let Claude handle complexity**: Use Claude commands for operations involving multiple branches
2. **Use regular commands for navigation**: Simple operations like $$stackit log$$ don't need AI assistance
3. **Review suggestions**: Claude will explain what it's doing—review before proceeding
4. **Leverage context awareness**: Claude commands understand your stack structure and can make intelligent decisions

## Troubleshooting

If Claude commands aren't working:

1. Verify installation:
   ```bash
   stackit agent install
   ```

2. Check for updates:
   ```bash
   brew upgrade stackit
   ```

3. Consult the [troubleshooting guide](../guide/troubleshooting.md)

## Next steps

- [View all CLI commands →](../cli/reference.md)
- [Learn common workflows →](../guide/workflows.md)
- [Other integrations →](index.md)
