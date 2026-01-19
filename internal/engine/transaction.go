package engine

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"stackit.dev/stackit/internal/git"
)

// Transaction retry configuration
const (
	// MaxRetries is the maximum number of retry attempts for transactional operations.
	// 5 retries with exponential backoff gives ~3 seconds total wait time.
	MaxRetries = 5

	// RetryBaseDelay is the base delay between retry attempts.
	// With exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms (capped).
	RetryBaseDelay = 100 * time.Millisecond

	// maxBackoffExponent caps exponential backoff to prevent excessive delays.
	// 2^4 * 100ms = 1.6s maximum base delay before jitter.
	maxBackoffExponent = 4
)

// Sentinel errors for transaction operations
var (
	// ErrTransactionCommitted is returned when attempting to modify a committed transaction
	ErrTransactionCommitted = errors.New("transaction already committed")
	// ErrTransactionRolledBack is returned when attempting to use a rolled-back transaction
	ErrTransactionRolledBack = errors.New("transaction was rolled back")
)

// MetadataTx represents an atomic metadata transaction.
// It batches multiple metadata updates and commits them atomically using
// git update-ref --stdin. All updates either succeed together or fail together.
type MetadataTx struct {
	eng *engineImpl
	mu  sync.Mutex

	// Staged changes
	metaUpdates  map[string]*git.Meta
	localUpdates map[string]*git.LocalMeta

	// Original ref SHAs for compare-and-swap validation
	originalMeta      map[string]string
	originalLocalMeta map[string]string

	// Transaction metadata
	message    string
	committed  bool
	rolledBack bool
}

// BeginTx starts a new metadata transaction with a descriptive message.
// The message is used for reflog entries and debugging.
func (e *engineImpl) BeginTx(message string) *MetadataTx {
	return &MetadataTx{
		eng:               e,
		metaUpdates:       make(map[string]*git.Meta),
		localUpdates:      make(map[string]*git.LocalMeta),
		originalMeta:      make(map[string]string),
		originalLocalMeta: make(map[string]string),
		message:           message,
	}
}

// UpdateMeta stages a metadata update for atomic commit.
// The update is not applied until Commit() is called.
func (tx *MetadataTx) UpdateMeta(branch string, meta *git.Meta) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return ErrTransactionCommitted
	}
	if tx.rolledBack {
		return ErrTransactionRolledBack
	}

	// Capture original SHA on first access for CAS validation
	if _, exists := tx.originalMeta[branch]; !exists {
		tx.originalMeta[branch] = tx.eng.git.GetMetadataRefSHA(branch)
	}

	tx.metaUpdates[branch] = meta
	return nil
}

// UpdateLocalMeta stages a local metadata update for atomic commit.
func (tx *MetadataTx) UpdateLocalMeta(branch string, meta *git.LocalMeta) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return ErrTransactionCommitted
	}
	if tx.rolledBack {
		return ErrTransactionRolledBack
	}

	// Capture original SHA on first access
	if _, exists := tx.originalLocalMeta[branch]; !exists {
		tx.originalLocalMeta[branch] = tx.eng.git.GetLocalMetadataRefSHA(branch)
	}

	tx.localUpdates[branch] = meta
	return nil
}

// Commit atomically applies all staged metadata changes.
// All updates succeed together or fail together via git update-ref --stdin.
// On success, the in-memory cache is updated to reflect the changes.
func (tx *MetadataTx) Commit(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return ErrTransactionCommitted
	}
	if tx.rolledBack {
		return ErrTransactionRolledBack
	}

	if len(tx.metaUpdates) == 0 && len(tx.localUpdates) == 0 {
		tx.committed = true
		return nil
	}

	// Build batch of ref updates with deterministic ordering
	refUpdates := make([]git.RefUpdate, 0, len(tx.metaUpdates)+len(tx.localUpdates))

	// Sort branch names for deterministic order (makes debugging easier)
	metaBranches := make([]string, 0, len(tx.metaUpdates))
	for branch := range tx.metaUpdates {
		metaBranches = append(metaBranches, branch)
	}
	slices.Sort(metaBranches)

	for _, branch := range metaBranches {
		meta := tx.metaUpdates[branch]
		blobSHA, err := tx.eng.git.WriteMetadataBlob(meta)
		if err != nil {
			return fmt.Errorf("write blob for %s: %w", branch, err)
		}

		refUpdates = append(refUpdates, git.RefUpdate{
			RefName: git.MetadataRefName(branch),
			NewSHA:  blobSHA,
			OldSHA:  tx.originalMeta[branch], // CAS validation
		})
	}

	// Sort local metadata branches for deterministic order
	localBranches := make([]string, 0, len(tx.localUpdates))
	for branch := range tx.localUpdates {
		localBranches = append(localBranches, branch)
	}
	slices.Sort(localBranches)

	for _, branch := range localBranches {
		meta := tx.localUpdates[branch]
		blobSHA, err := tx.eng.git.WriteLocalMetadataBlob(meta)
		if err != nil {
			return fmt.Errorf("write local blob for %s: %w", branch, err)
		}

		refUpdates = append(refUpdates, git.RefUpdate{
			RefName: git.LocalMetadataRefName(branch),
			NewSHA:  blobSHA,
			OldSHA:  tx.originalLocalMeta[branch],
		})
	}

	// Atomic batch update with reflog message
	if err := tx.eng.git.UpdateRefsBatchWithLog(ctx, refUpdates, tx.message); err != nil {
		return fmt.Errorf("atomic commit failed: %w", err)
	}

	// Update in-memory cache (using same sorted order for consistency)
	tx.eng.mu.Lock()
	for _, branch := range metaBranches {
		tx.eng.updateBranchStateFromMeta(branch, tx.metaUpdates[branch])
	}
	for _, branch := range localBranches {
		tx.eng.updateBranchStateFromLocalMeta(branch, tx.localUpdates[branch])
	}
	tx.eng.mu.Unlock()

	tx.committed = true
	return nil
}

// Rollback discards all staged changes without applying them.
func (tx *MetadataTx) Rollback() {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	tx.metaUpdates = nil
	tx.localUpdates = nil
	tx.rolledBack = true
}

// IsCommitted returns true if the transaction has been committed.
func (tx *MetadataTx) IsCommitted() bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.committed
}

// updateBranchStateFromMeta updates the in-memory branch state from metadata.
//
// IMPORTANT: Caller MUST hold e.mu lock before calling this function.
// This function modifies e.childrenMap and e.branchState which are not
// individually thread-safe. Failing to hold the lock can cause data races
// and corrupt the in-memory cache.
func (e *engineImpl) updateBranchStateFromMeta(branch string, meta *git.Meta) {
	state := e.branchState.GetOrCreate(branch)

	if meta.ParentBranchName != nil {
		// Update children map for old parent
		if state.Parent != "" && state.Parent != *meta.ParentBranchName {
			oldChildren := e.childrenMap[state.Parent]
			for i, c := range oldChildren {
				if c == branch {
					e.childrenMap[state.Parent] = append(oldChildren[:i], oldChildren[i+1:]...)
					break
				}
			}
		}

		state.Parent = *meta.ParentBranchName

		// Update children map for new parent
		if state.Parent != "" {
			if !slices.Contains(e.childrenMap[state.Parent], branch) {
				e.childrenMap[state.Parent] = append(e.childrenMap[state.Parent], branch)
			}
		}
	}

	if meta.Scope != nil {
		state.Scope = *meta.Scope
	} else {
		state.Scope = ""
	}

	state.LockReason = meta.LockReason
	state.BranchType = meta.BranchType
}

// updateBranchStateFromLocalMeta updates the in-memory branch state from local metadata.
//
// IMPORTANT: Caller MUST hold e.mu lock before calling this function.
// This function modifies e.branchState which is not individually thread-safe.
func (e *engineImpl) updateBranchStateFromLocalMeta(branch string, meta *git.LocalMeta) {
	state := e.branchState.GetOrCreate(branch)
	state.Frozen = meta.Frozen
}

// IsConcurrentModificationError returns true if the error indicates a concurrent
// modification conflict (CAS failure). This includes various git update-ref errors
// that occur when references are modified between read and write operations.
func IsConcurrentModificationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Standard git update-ref CAS failures
	return strings.Contains(errStr, "cannot lock ref") ||
		strings.Contains(errStr, "reference is not expected") ||
		strings.Contains(errStr, "expected old-value") ||
		// Additional git lock contention errors
		strings.Contains(errStr, "stale info") ||
		strings.Contains(errStr, "Unable to create") && strings.Contains(errStr, ".lock")
}

// WithRetry executes an operation with automatic retry on concurrent modification errors.
// It uses exponential backoff with jitter to prevent thundering herd.
func (e *engineImpl) WithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := range MaxRetries {
		if err := operation(); err != nil {
			if IsConcurrentModificationError(err) {
				lastErr = err

				// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms base
				baseDelay := RetryBaseDelay * time.Duration(1<<min(attempt, maxBackoffExponent))
				// Add jitter: +/- 25% of base delay to prevent thundering herd
				// Using math/rand is fine here - jitter doesn't need cryptographic security
				jitter := time.Duration(rand.Int64N(int64(baseDelay / 2))) //nolint:gosec
				delay := baseDelay - baseDelay/4 + jitter

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}

				// Refresh cache before retry.
				// We ignore rebuild errors because the operation itself might still succeed,
				// and if it doesn't, it will fail with a more specific error.
				_ = e.rebuild()
				continue
			}
			return err // Non-retryable error
		}
		return nil // Success
	}

	return fmt.Errorf("operation failed after %d retries: %w", MaxRetries, lastErr)
}
