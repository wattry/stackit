---
icon: material/sitemap
---

# Workflows

Practical guides for using stackit effectively in your daily development.

<div class="grid cards" markdown>

-   :material-clock-outline:{ .lg .middle } **Daily Workflows**

    ---

    Common tasks like updating after code review, syncing with main, and using absorb for quick fixes.

    [Daily workflows →](daily.md)

-   :material-tools:{ .lg .middle } **Advanced Workflows**

    ---

    Power-user operations: splitting commits, reorganizing stacks, moving branches, and running commands across the stack.

    [Advanced workflows →](advanced.md)

-   :material-account-group:{ .lg .middle } **Team Collaboration**

    ---

    Sharing configuration, protecting branches, fetching teammate stacks, and CI integration.

    [Collaboration →](collaboration.md)

</div>

## Quick Reference

| Task | Command |
|------|---------|
| Update after code review | `stackit modify` then `stackit restack` |
| Absorb fixes into correct branches | `stackit absorb` |
| Sync with main | `stackit sync` |
| Split commits into branches | `stackit split` |
| Move branch to new parent | `stackit move <branch> <new-parent>` |
| Fetch teammate's stack | `stackit get <pr-number>` |

## Related

- [Core Concepts →](../guide/concepts.md)
- [Worktrees →](../worktrees/index.md)
- [CLI Reference →](../cli/reference.md)
