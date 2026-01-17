# Suggest Next Command

After completing a stackit operation, use `AskUserQuestion` to offer relevant follow-up commands.

## Follow-up Matrix

| Just Completed | Primary Option | Secondary Option | Condition |
|---------------|----------------|------------------|-----------|
| `create` | `/stack-submit` | `/stack-create` | Always |
| `submit` | `/stack-sync` | `stackit merge next` | Always |
| `sync` | `/stack-submit` | View status | If branches remain |
| `restack` | `/stack-submit` | View stack | Always |
| `absorb` | `/stack-restack` | `/stack-submit` | Always |
| `fold` | `/stack-restack` | `/stack-submit` | Always |
| `verify` | `/stack-fix` | `/stack-submit` | If failures, else submit |
| `fix` | `/stack-submit` | `/stack-restack` | Always |
| `merge next` | `/stack-sync` | View stack | Always |

## Implementation Pattern

Use `AskUserQuestion` with:
- Header: "Next step"
- Question: Contextual question about what to do next
- Options:
  - Primary follow-up (recommended)
  - Secondary alternative
  - "Done for now"

If user selects a skill option, invoke that skill using the `Skill` tool.
If user selects "Done for now", end gracefully with a summary.

## Example

After `stack-create` completes successfully:

```
AskUserQuestion:
  header: "Next step"
  question: "Branch created successfully. What would you like to do next?"
  options:
    - label: "Submit as PR (Recommended)"
      description: "Push branch and create/update pull request"
    - label: "Stack another change"
      description: "Create another branch on top of this one"
    - label: "Done for now"
      description: "No follow-up action needed"
```

## Response Handling

- **"Submit as PR"**: Invoke `/stack-submit` skill
- **"Stack another change"**: Tell user to stage changes and run `/stack-create`
- **"Sync with trunk"**: Invoke `/stack-sync` skill
- **"Restack branches"**: Invoke `/stack-restack` skill
- **"Fix issues"**: Invoke `/stack-fix` skill
- **"View stack"**: Run `command stackit log --no-interactive`
- **"Done for now"**: End with summary of current state
