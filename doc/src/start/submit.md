---
icon: material/upload
---

# Submit your PRs

Once you have a stack of branches, it's time to create Pull Requests on GitHub.

## Submit the entire stack

```bash
stackit submit
```

This command:

1. Pushes all branches to GitHub
2. Creates PRs for each branch
3. Sets the correct base branch for each PR (child branches point to their parent)
4. Generates PR descriptions with stack context

## Submit options

### Submit only the current branch

```bash
stackit submit --branch
```

### Submit as draft PRs

```bash
stackit submit --draft
```

### Submit the current stack

```bash
stackit submit --stack
```

Or use the shorthand alias:

```bash
stackit ss  # Equivalent to: stackit submit --stack
```

## PR descriptions

Stackit generates PR descriptions that include:

- Your commit message
- Position in the stack
- Links to parent and child PRs
- Visual representation of the stack structure

You can customize the footer behavior with:

```bash
stackit config set submit.footer true
```

## Updating PRs

After making changes to your stack:

1. Make your changes and commit:
   ```bash
   stackit modify  # Amend the current commit
   ```

2. Restack child branches:
   ```bash
   stackit restack
   ```

3. Update the PRs:
   ```bash
   stackit submit
   ```

Stackit will update existing PRs instead of creating duplicates.

## Merge your stack

Once your PRs are approved, merge the entire stack:

### Interactive merge wizard

```bash
stackit merge
```

Launches an interactive wizard to guide you through merging options.

### Merge bottom PR, then restack

```bash
stackit merge next
```

Merges the bottom-most unmerged PR using GitHub automerge, then restacks remaining branches.

### Consolidate and merge

```bash
stackit merge squash
```

Consolidates all branches into a single PR for atomic merging.

## Next steps

- [Learn about common workflows →](../guide/workflows.md)
- [Understand core concepts →](../guide/concepts.md)
- [Explore the CLI reference →](../cli/reference.md)
