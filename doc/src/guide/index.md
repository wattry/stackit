---
icon: material/book-open-variant
---

# User Guide

This guide covers everything you need to know to work effectively with stackit.

## Topics

<div class="grid cards" markdown>

-   :material-graph:{ .lg .middle } **Core Concepts**

    ---

    Learn about stacks, parents, children, and how stackit manages branch relationships.

    [Learn concepts →](concepts.md)

-   :material-workflow:{ .lg .middle } **Common Workflows**

    ---

    Real-world examples of using stackit for code reviews, updates, and collaboration.

    [View workflows →](workflows.md)

-   :material-folder-multiple:{ .lg .middle } **Worktrees**

    ---

    Work on multiple stacks in parallel, each in its own directory.

    [Learn worktrees →](worktrees.md)

-   :material-account-group:{ .lg .middle } **Team Workflows**

    ---

    Share configuration, protect branches, and collaborate on stacks with your team.

    [Team workflows →](team-workflows.md)

-   :material-help-circle:{ .lg .middle } **Troubleshooting**

    ---

    Solutions to common issues and error messages.

    [Get help →](troubleshooting.md)

</div>

## Quick reference

### Essential commands

- $$stackit log$$ - View your stack
- $$stackit create$$ - Create a new branch
- $$stackit submit$$ - Create/update PRs
- $$stackit sync$$ - Update from trunk
- $$stackit merge$$ - Merge your stack

### Need help fast?

Run the doctor command to diagnose common issues:

```bash
stackit doctor
```

Or check the [FAQ](../community/faq.md) for frequently asked questions.

### Integrations

Looking to integrate stackit with your tools? See the [Integrations](../integrations/index.md) section for Claude Code, GitHub Actions, git hooks, and shell integration.
