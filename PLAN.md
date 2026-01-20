# Metadata Storage Architecture Improvements

## Executive Summary

The core improvements are:

1. **Transaction API** - Atomic multi-branch metadata updates with rollback
2. **Retry Logic** - Handle concurrent modification conflicts automatically
3. **Audit Trail** - Optional commit-based history for debugging
4. **Performance Optimizations** - Move disk I/O outside critical sections

## Current State Analysis

### Strengths to Preserve
- **Per-branch refs**: Fast single-branch operations (`refs/stackit/metadata/<branch>`)
- **Parallel batch reads**: `BatchReadMetadata()` with goroutines
- **Atomic ref updates**: `UpdateRefsBatch()` using `git update-ref --stdin`
- **Separated local state**: `refs/stackit/local-metadata/` for non-synced data
- **In-memory caching**: `BranchStateMap` for O(1) lookups

### Critical Gaps
- **No transaction layer**: Individual writes can leave inconsistent state
- **Lock contention**: Disk I/O happens inside `e.mu.Lock()` critical section
- **No retry logic**: Concurrent modifications fail without recovery
- **No audit trail**: Metadata changes overwrite previous state
- **Complex sync**: Manual conflict resolution required

---

## Phase 1: Transaction API (High Priority)

### Goal
Implement atomic multi-branch metadata updates with explicit transaction boundaries.

### Design

```go
// internal/engine/transaction.go

type MetadataTx struct {
    eng       *engineImpl
    updates   map[string]*git.Meta       // branch -> updated metadata
    localUpdates map[string]*git.LocalMeta
    message   string                     // commit message for audit trail
    committed bool
}

// Begin a new transaction
func (e *engineImpl) BeginTx(message string) *MetadataTx

// Stage an update (doesn't write to disk yet)
func (tx *MetadataTx) UpdateMeta(branch string, meta *git.Meta) error
func (tx *MetadataTx) UpdateLocalMeta(branch string, meta *git.LocalMeta) error

// Commit all changes atomically
func (tx *MetadataTx) Commit(ctx context.Context) error

// Rollback discards all staged changes
func (tx *MetadataTx) Rollback()
```

### Implementation Steps

#### 1.1 Create Transaction Struct
**File**: `internal/engine/transaction.go` (new)

```go
package engine

import (
    "context"
    "fmt"
    "sync"

    "stackit.dev/stackit/internal/git"
)

type MetadataTx struct {
    eng          *engineImpl
    mu           sync.Mutex

    // Staged changes
    metaUpdates  map[string]*git.Meta
    localUpdates map[string]*git.LocalMeta

    // Original state for rollback validation
    originalMeta map[string]string  // branch -> original ref SHA

    // Transaction metadata
    message   string
    committed bool
    rolledBack bool
}

func (e *engineImpl) BeginTx(message string) *MetadataTx {
    return &MetadataTx{
        eng:          e,
        metaUpdates:  make(map[string]*git.Meta),
        localUpdates: make(map[string]*git.LocalMeta),
        originalMeta: make(map[string]string),
        message:      message,
    }
}
```

#### 1.2 Implement Staging Methods
```go
func (tx *MetadataTx) UpdateMeta(branch string, meta *git.Meta) error {
    tx.mu.Lock()
    defer tx.mu.Unlock()

    if tx.committed || tx.rolledBack {
        return fmt.Errorf("transaction already finished")
    }

    // Capture original SHA for conflict detection
    if _, exists := tx.originalMeta[branch]; !exists {
        sha, _ := tx.eng.git.ReadMetadataRefSHA(branch)
        tx.originalMeta[branch] = sha
    }

    tx.metaUpdates[branch] = meta
    return nil
}
```

#### 1.3 Implement Atomic Commit
```go
func (tx *MetadataTx) Commit(ctx context.Context) error {
    tx.mu.Lock()
    defer tx.mu.Unlock()

    if tx.committed {
        return fmt.Errorf("transaction already committed")
    }
    if tx.rolledBack {
        return fmt.Errorf("transaction was rolled back")
    }

    // Build batch of ref updates
    var refUpdates []git.RefUpdate

    for branch, meta := range tx.metaUpdates {
        // Create blob for metadata
        blobSHA, err := tx.eng.git.WriteMetadataBlob(meta)
        if err != nil {
            return fmt.Errorf("write blob for %s: %w", branch, err)
        }

        refUpdates = append(refUpdates, git.RefUpdate{
            Ref:    git.MetadataRef(branch),
            NewSHA: blobSHA,
            OldSHA: tx.originalMeta[branch], // For CAS validation
        })
    }

    for branch, meta := range tx.localUpdates {
        blobSHA, err := tx.eng.git.WriteLocalMetadataBlob(meta)
        if err != nil {
            return fmt.Errorf("write local blob for %s: %w", branch, err)
        }

        refUpdates = append(refUpdates, git.RefUpdate{
            Ref:    git.LocalMetadataRef(branch),
            NewSHA: blobSHA,
        })
    }

    // Atomic batch update with message
    if err := tx.eng.git.UpdateRefsBatchWithLog(ctx, refUpdates, tx.message); err != nil {
        return fmt.Errorf("atomic commit failed: %w", err)
    }

    // Update in-memory cache
    tx.eng.mu.Lock()
    for branch, meta := range tx.metaUpdates {
        tx.eng.updateBranchStateFromMeta(branch, meta)
    }
    for branch, meta := range tx.localUpdates {
        tx.eng.updateBranchStateFromLocalMeta(branch, meta)
    }
    tx.eng.mu.Unlock()

    tx.committed = true
    return nil
}
```

#### 1.4 Add Compare-and-Swap Support to git.UpdateRefsBatch
**File**: `internal/git/runner.go`

Modify `UpdateRefsBatch` to support optional old SHA validation:

```go
type RefUpdate struct {
    Ref    string
    NewSHA string
    OldSHA string // Optional: if set, validates current SHA before update
}

func (r *Runner) UpdateRefsBatch(ctx context.Context, updates []RefUpdate) error {
    // Build stdin commands
    var commands []string
    for _, u := range updates {
        if u.OldSHA != "" {
            // Compare-and-swap: update ref newvalue oldvalue
            commands = append(commands, fmt.Sprintf("update %s %s %s", u.Ref, u.NewSHA, u.OldSHA))
        } else {
            commands = append(commands, fmt.Sprintf("update %s %s", u.Ref, u.NewSHA))
        }
    }

    // ... execute via git update-ref --stdin
}
```

#### 1.5 Migrate Existing Operations to Use Transactions
**Priority operations to migrate**:

1. `SetLocked()` - Updates multiple branches
2. `RestackBranches()` - Updates parent relationships across stack
3. `DeleteBranches()` - Removes branches and their metadata
4. `SetScope()` with propagation - Updates scope for branch and descendants

**Example migration** (`internal/engine/engine_writer.go`):

```go
// Before
func (e *engineImpl) SetLocked(branches []Branch, reason LockReason) {
    e.mu.Lock()
    defer e.mu.Unlock()
    for _, branch := range branches {
        meta, _ := e.git.ReadMetadata(branch.Name)
        meta.LockReason = reason
        e.git.WriteMetadata(branch.Name, meta)
    }
}

// After
func (e *engineImpl) SetLocked(ctx context.Context, branches []Branch, reason LockReason) error {
    tx := e.BeginTx(fmt.Sprintf("lock: set %s on %d branches", reason, len(branches)))

    // Batch read all metadata first (parallel, outside any lock)
    metas, err := e.git.BatchReadMetadata(branchNames(branches))
    if err != nil {
        return err
    }

    // Stage all updates
    for _, branch := range branches {
        meta := metas[branch.Name]
        if meta == nil {
            meta = &git.Meta{}
        }
        meta.LockReason = reason
        tx.UpdateMeta(branch.Name, meta)
    }

    // Atomic commit
    return tx.Commit(ctx)
}
```

### Testing Strategy
- Unit tests for transaction commit/rollback
- Integration tests for concurrent transactions
- Test CAS failure detection and error messages

---

## Phase 2: Retry Logic (High Priority)

### Goal
Automatically retry metadata operations when concurrent modifications cause conflicts.

### Design

```go
// internal/engine/retry.go

const (
    MaxRetries    = 5
    RetryBaseDelay = 100 * time.Millisecond
)

func (e *engineImpl) WithRetry(ctx context.Context, operation func() error) error {
    var lastErr error
    for attempt := 0; attempt < MaxRetries; attempt++ {
        if err := operation(); err != nil {
            if isConcurrentModificationError(err) {
                lastErr = err
                // Exponential backoff with jitter
                delay := RetryBaseDelay * time.Duration(1<<attempt)
                jitter := time.Duration(rand.Int63n(int64(delay / 4)))
                time.Sleep(delay + jitter)

                // Refresh cache before retry
                e.rebuild()
                continue
            }
            return err // Non-retryable error
        }
        return nil
    }
    return fmt.Errorf("operation failed after %d retries: %w", MaxRetries, lastErr)
}

func isConcurrentModificationError(err error) bool {
    // Detect git update-ref CAS failures
    return strings.Contains(err.Error(), "cannot lock ref") ||
           strings.Contains(err.Error(), "reference is not expected")
}
```

### Implementation Steps

#### 2.1 Create Retry Wrapper
**File**: `internal/engine/retry.go` (new)

#### 2.2 Detect CAS Failures
Parse git update-ref error messages to identify concurrent modification.

#### 2.3 Wrap Critical Operations
```go
func (e *engineImpl) SetLocked(ctx context.Context, branches []Branch, reason LockReason) error {
    return e.WithRetry(ctx, func() error {
        tx := e.BeginTx("lock branches")
        // ... stage updates
        return tx.Commit(ctx)
    })
}
```

### Testing Strategy
- Simulate concurrent modifications with goroutines
- Test exponential backoff timing
- Test max retry limit

---

## Phase 3: Audit Trail (Medium Priority)

### Goal
Record metadata changes for debugging and potential undo without separate snapshot files.

### Design Options

#### Option A: Hybrid Ref + History Commit (Recommended)
Keep per-branch refs for fast access, but also maintain a history commit tree for audit.

```
refs/stackit/metadata/feature-1 → blob abc123 (current state)
refs/stackit/metadata/feature-2 → blob def456 (current state)
refs/stackit/history → commit xyz789 (audit trail)
  └── tree:
        └── log/2024-01-15T10:30:00-lock.json → "locked feature-1, feature-2"
        └── log/2024-01-15T10:35:00-restack.json → "restacked stack"
```

#### Option B: Ref History via Reflogs
Configure git to maintain reflogs for stackit refs:

```bash
git config core.logAllRefUpdates always
```

Then inspect via:
```bash
git reflog refs/stackit/metadata/feature-1
```

### Implementation (Option A)

#### 3.1 Create History Entry on Commit
**File**: `internal/engine/transaction.go`

```go
func (tx *MetadataTx) Commit(ctx context.Context) error {
    // ... existing commit logic ...

    // Record audit entry
    if tx.eng.auditEnabled {
        tx.eng.recordAuditEntry(ctx, tx.message, tx.metaUpdates)
    }

    tx.committed = true
    return nil
}

func (e *engineImpl) recordAuditEntry(ctx context.Context, message string, changes map[string]*git.Meta) error {
    entry := AuditEntry{
        Timestamp: time.Now(),
        Message:   message,
        Changes:   summarizeChanges(changes),
    }

    entryJSON, _ := json.Marshal(entry)
    blobSHA, _ := e.git.HashObject(entryJSON)

    // Add to history tree and commit
    return e.git.AppendToHistoryTree(ctx, blobSHA, entry.Timestamp)
}
```

#### 3.2 Audit Query API
```go
func (e *engineImpl) GetAuditLog(ctx context.Context, limit int) ([]AuditEntry, error)
func (e *engineImpl) GetBranchHistory(ctx context.Context, branch string) ([]AuditEntry, error)
```

### Testing Strategy
- Verify audit entries created on commit
- Test audit log querying
- Test filtering by branch

---

## Phase 4: Performance Optimizations (Medium Priority)

### Goal
Move disk I/O outside critical lock sections.

### Current Problem
```go
func (e *engineImpl) SetLocked(...) {
    e.mu.Lock()                          // Lock acquired
    for _, branch := range branches {
        meta, _ := e.git.ReadMetadata()  // DISK I/O IN LOCK
        meta.LockReason = reason
        e.git.WriteMetadata()            // DISK I/O IN LOCK
    }
    e.mu.Unlock()                        // Lock released
}
```

### Solution Pattern
```go
func (e *engineImpl) SetLocked(...) error {
    // Phase 1: Read (parallel, no lock)
    metas, err := e.git.BatchReadMetadata(branchNames(branches))
    if err != nil {
        return err
    }

    // Phase 2: Prepare updates (no disk I/O)
    updates := make(map[string]*git.Meta)
    for _, branch := range branches {
        meta := metas[branch.Name]
        meta.LockReason = reason
        updates[branch.Name] = meta
    }

    // Phase 3: Write (atomic batch, brief lock for cache update)
    tx := e.BeginTx("lock branches")
    for branch, meta := range updates {
        tx.UpdateMeta(branch, meta)
    }
    return tx.Commit(ctx)  // Lock only held for cache update
}
```

### Implementation Steps

#### 4.1 Review All Mutation Methods
Identify methods that do disk I/O inside lock:
- `SetLocked()` ✓
- `SetFrozen()` ✓
- `SetScope()` ✓
- `SetParent()` ✓
- `UpsertPrInfo()` ✓
- `DeleteBranchMetadata()` ✓

#### 4.2 Refactor to Read-Prepare-Write Pattern
For each method:
1. Batch read metadata outside lock
2. Prepare updates
3. Commit via transaction (minimal lock time)

#### 4.3 Add Batch Write Helpers
**File**: `internal/git/metadata.go`

```go
func (r *Runner) WriteMetadataBlob(meta *Meta) (string, error) {
    data, _ := json.Marshal(meta)
    return r.HashObject(data)  // Returns blob SHA without updating ref
}

func (r *Runner) BatchWriteMetadata(ctx context.Context, updates map[string]*Meta) error {
    var refUpdates []RefUpdate
    for branch, meta := range updates {
        blobSHA, err := r.WriteMetadataBlob(meta)
        if err != nil {
            return err
        }
        refUpdates = append(refUpdates, RefUpdate{
            Ref:    MetadataRef(branch),
            NewSHA: blobSHA,
        })
    }
    return r.UpdateRefsBatch(ctx, refUpdates)
}
```

---

## Phase 5: Remote Sync Improvements (Low Priority)

### Goal
Simplify remote sync with better conflict resolution.

### Current Issues
- Manual refspec configuration required
- Conflict resolution requires user interaction
- No automatic retry for transient failures

### Improvements

#### 5.1 Automatic Refspec Setup
```go
func (e *engineImpl) SetupRemoteSync(ctx context.Context, remote string) error {
    // Configure fetch refspec
    e.git.ConfigSet(ctx, fmt.Sprintf("remote.%s.fetch", remote),
        "+refs/stackit/metadata/*:refs/stackit/remote-metadata/%s/*", remote)

    // Configure push refspec
    e.git.ConfigSet(ctx, fmt.Sprintf("remote.%s.push", remote),
        "refs/stackit/metadata/*:refs/stackit/metadata/*")

    return nil
}
```

#### 5.2 Auto-Resolution for Non-Conflicting Changes
```go
func (e *engineImpl) SyncMetadata(ctx context.Context) error {
    diffs := e.ComputeMetadataDiff()

    for _, diff := range diffs {
        switch diff.Type {
        case DiffTypeRemoteOnly:
            // Remote has changes, local unchanged → accept remote
            e.AcceptRemoteMetadata(diff.Branch)

        case DiffTypeLocalOnly:
            // Local has changes, remote unchanged → keep local
            continue

        case DiffTypeBothChanged:
            // Both changed → require user intervention
            conflicts = append(conflicts, diff)
        }
    }

    return conflicts
}
```

---

## Implementation Order

### Sprint 1: Foundation (Phase 1.1-1.4)
- [ ] Create `MetadataTx` struct
- [ ] Implement staging methods
- [ ] Add CAS support to `UpdateRefsBatch`
- [ ] Implement atomic commit
- [ ] Write unit tests

### Sprint 2: Migration (Phase 1.5 + 4)
- [ ] Migrate `SetLocked()` to transactions
- [ ] Migrate `SetFrozen()` to transactions
- [ ] Migrate `SetScope()` to transactions
- [ ] Refactor to read-prepare-write pattern
- [ ] Integration tests for concurrent operations

### Sprint 3: Reliability (Phase 2)
- [ ] Implement retry wrapper
- [ ] Add CAS failure detection
- [ ] Wrap critical operations
- [ ] Stress tests for concurrent modifications

### Sprint 4: Observability (Phase 3)
- [ ] Design audit entry format
- [ ] Implement audit recording
- [ ] Add audit query API
- [ ] CLI commands for viewing history

### Sprint 5: Polish (Phase 5)
- [ ] Auto-setup refspecs
- [ ] Improve conflict detection
- [ ] Auto-resolution for non-conflicts
- [ ] Documentation updates

---

## Migration Strategy

### Backward Compatibility
- Existing metadata refs remain unchanged
- New transaction layer is internal; external API unchanged
- Audit trail is additive; doesn't modify existing refs

### Gradual Rollout
1. Add transaction layer without changing existing code paths
2. Migrate one operation at a time (SetLocked first)
3. Run both paths in parallel with validation
4. Remove old code paths once validated

### Testing Requirements
- All existing tests must pass
- Add new tests for transaction semantics
- Benchmark performance before/after
- Stress test concurrent operations

---

## Success Metrics

1. **Atomicity**: Multi-branch operations either fully succeed or fully fail
2. **Performance**: Lock contention reduced by 50%+
3. **Reliability**: Zero inconsistent states from concurrent operations
4. **Debuggability**: Full audit trail of metadata changes

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| CAS failures increase with adoption | Medium | Retry logic handles gracefully |
| Audit trail bloat | Low | Configurable, can prune old entries |
| Git update-ref --stdin edge cases | Medium | Extensive integration testing |
| Migration breaks existing stacks | High | Backward compatible, gradual rollout |

---

## Open Questions

1. **Audit trail storage**: Should we use a single history ref or per-branch reflogs?
2. **Retry backoff**: What's the optimal base delay and jitter range?
3. **Transaction timeout**: Should transactions have a maximum lifetime?
4. **Cache invalidation**: Should we use fine-grained invalidation or full rebuild after tx?
