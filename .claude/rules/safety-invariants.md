# Safety Invariants

Critical guarantees that all operations must maintain. These are non-negotiable.

## No Detached HEAD State

**Operations must NEVER leave the user in a detached HEAD state when cancelled or on failure.**

This applies to: split, sync, create, merge, absorb, fold, restack, and any other operation that modifies branch state.

### Implementation Pattern

1. Capture the current branch at the start of the operation
2. Take a snapshot before any mutations: `eng.TakeSnapshot(...)`
3. Perform the operation
4. On error or cancellation, restore the original branch before returning

```go
// Example pattern
currentBranch := eng.CurrentBranch()
if currentBranch == nil {
    return fmt.Errorf("not on a branch")
}

// ... operation logic ...

// On cancellation/error:
if err := git.CheckoutBranch(currentBranch.Name); err != nil {
    // Log but don't mask original error
}
return originalErr
```

### Why This Matters

- Users expect to be on a branch after any operation completes (success or failure)
- Detached HEAD is confusing and requires manual recovery
- Cancelled operations should have minimal side effects
