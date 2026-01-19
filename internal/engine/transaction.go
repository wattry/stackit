package engine

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"stackit.dev/stackit/internal/git"
)

// Transaction retry configuration
const (
	// MaxRetries is the maximum number of retry attempts for transactional operations
	MaxRetries = 5
	// RetryBaseDelay is the base delay between retry attempts
	RetryBaseDelay = 100 * time.Millisecond
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
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction was rolled back")
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
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction was rolled back")
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
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction was rolled back")
	}

	if len(tx.metaUpdates) == 0 && len(tx.localUpdates) == 0 {
		tx.committed = true
		return nil
	}

	// Build batch of ref updates
	refUpdates := make([]git.RefUpdate, 0, len(tx.metaUpdates)+len(tx.localUpdates))

	for branch, meta := range tx.metaUpdates {
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

	for branch, meta := range tx.localUpdates {
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
// Caller must hold e.mu lock.
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
// Caller must hold e.mu lock.
func (e *engineImpl) updateBranchStateFromLocalMeta(branch string, meta *git.LocalMeta) {
	state := e.branchState.GetOrCreate(branch)
	state.Frozen = meta.Frozen
}

// IsConcurrentModificationError returns true if the error indicates a concurrent
// modification conflict (CAS failure).
func IsConcurrentModificationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "cannot lock ref") ||
		strings.Contains(errStr, "reference is not expected") ||
		strings.Contains(errStr, "expected old-value")
}

// WithRetry executes an operation with automatic retry on concurrent modification errors.
// It uses exponential backoff with jitter.
func (e *engineImpl) WithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := range MaxRetries {
		if err := operation(); err != nil {
			if IsConcurrentModificationError(err) {
				lastErr = err

				// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
				// Use attempt as a simple "jitter" to spread out retries
				delay := RetryBaseDelay * time.Duration(1<<min(attempt, 4))

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}

				// Refresh cache before retry
				_ = e.rebuild()
				continue
			}
			return err // Non-retryable error
		}
		return nil // Success
	}

	return fmt.Errorf("operation failed after %d retries: %w", MaxRetries, lastErr)
}
