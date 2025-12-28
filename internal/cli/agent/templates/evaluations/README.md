# Stackit Skill Evaluations

This directory contains evaluation scenarios for testing the stackit Claude Code skill.

## Purpose

These evaluations help ensure the skill:
- Guides Claude to use stackit correctly
- Generates appropriate commit messages and PR descriptions
- Handles errors and edge cases properly
- Follows project conventions and best practices

## Evaluation Format

Each evaluation is a JSON file with:
- `skills`: List of skills required (e.g., ["stackit"])
- `query`: The user's request/question
- `setup_commands`: Commands to run before the test (optional)
- `expected_behavior`: List of behaviors that should occur

## Running Evaluations

These are reference scenarios for manual testing or custom evaluation frameworks.

```bash
# Example workflow for manual evaluation:
# 1. Load the stackit skill in Claude Code
# 2. Run setup commands
# 3. Make the query
# 4. Verify expected behaviors occur
```

## Evaluation Categories

- `create-branch.json` - Branch creation workflows
- `submit-pr.json` - PR submission and generation
- `fix-absorb.json` - Post-absorb error recovery
- `conflict-resolution.json` - Handling rebase conflicts
- `commit-messages.json` - Message generation quality
