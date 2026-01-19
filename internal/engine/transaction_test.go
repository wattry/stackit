package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestSetLocked_SingleBranch(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create a branch to test with
	s.CreateBranch("feature-1").
		Commit("feature 1 change")

	// Track the branch so it has metadata
	err := s.Engine.TrackBranch(context.Background(), "feature-1", "main")
	require.NoError(t, err)

	branch := s.Engine.GetBranch("feature-1")

	// Lock the branch
	result, err := s.Engine.SetLocked([]engine.Branch{branch}, engine.LockReasonUser)
	require.NoError(t, err)
	assert.Len(t, result.AffectedBranches, 1)
	assert.Contains(t, result.AffectedBranches, "feature-1")

	// Verify the change persisted
	impl := s.Engine.(interface{ Git() git.Runner })
	readMeta, err := impl.Git().ReadMetadata("feature-1")
	require.NoError(t, err)
	assert.Equal(t, git.LockReasonUser, readMeta.LockReason)

	// Verify in-memory cache is updated
	assert.True(t, branch.IsLocked())
}

func TestSetLocked_UsesTransaction(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create branches
	s.CreateBranch("feature-1").
		Commit("feature 1 change").
		CreateBranch("feature-2").
		Commit("feature 2 change")

	// Track branches
	err := s.Engine.TrackBranch(context.Background(), "feature-1", "main")
	require.NoError(t, err)
	err = s.Engine.TrackBranch(context.Background(), "feature-2", "feature-1")
	require.NoError(t, err)

	// Lock multiple branches
	result, err := s.Engine.SetLocked([]engine.Branch{
		s.Engine.GetBranch("feature-1"),
		s.Engine.GetBranch("feature-2"),
	}, engine.LockReasonUser)
	require.NoError(t, err)

	// Both should be affected
	assert.Len(t, result.AffectedBranches, 2)
	assert.Empty(t, result.Errors)

	// Verify persistence
	impl := s.Engine.(interface{ Git() git.Runner })
	meta1, err := impl.Git().ReadMetadata("feature-1")
	require.NoError(t, err)
	assert.Equal(t, git.LockReasonUser, meta1.LockReason)

	meta2, err := impl.Git().ReadMetadata("feature-2")
	require.NoError(t, err)
	assert.Equal(t, git.LockReasonUser, meta2.LockReason)
}

func TestSetFrozen_UsesTransaction(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create branches
	s.CreateBranch("feature-1").
		Commit("feature 1 change").
		CreateBranch("feature-2").
		Commit("feature 2 change")

	// Track branches
	err := s.Engine.TrackBranch(context.Background(), "feature-1", "main")
	require.NoError(t, err)
	err = s.Engine.TrackBranch(context.Background(), "feature-2", "feature-1")
	require.NoError(t, err)

	// Freeze multiple branches
	result, err := s.Engine.SetFrozen([]engine.Branch{
		s.Engine.GetBranch("feature-1"),
		s.Engine.GetBranch("feature-2"),
	}, true)
	require.NoError(t, err)

	// Both should be affected
	assert.Len(t, result.AffectedBranches, 2)
	assert.Empty(t, result.Errors)

	// Verify persistence
	impl := s.Engine.(interface{ Git() git.Runner })
	localMeta1, err := impl.Git().ReadLocalMetadata("feature-1")
	require.NoError(t, err)
	assert.True(t, localMeta1.Frozen)

	localMeta2, err := impl.Git().ReadLocalMetadata("feature-2")
	require.NoError(t, err)
	assert.True(t, localMeta2.Frozen)
}

func TestSetLocked_EmptyBranchList(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Empty branch list should succeed
	result, err := s.Engine.SetLocked([]engine.Branch{}, engine.LockReasonUser)
	require.NoError(t, err)
	assert.Empty(t, result.AffectedBranches)
	assert.Empty(t, result.Errors)
}

func TestSetFrozen_EmptyBranchList(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Empty branch list should succeed
	result, err := s.Engine.SetFrozen([]engine.Branch{}, true)
	require.NoError(t, err)
	assert.Empty(t, result.AffectedBranches)
	assert.Empty(t, result.Errors)
}

func TestSetLocked_UnlockBranches(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create and track a branch
	s.CreateBranch("feature-1").
		Commit("feature 1 change")
	err := s.Engine.TrackBranch(context.Background(), "feature-1", "main")
	require.NoError(t, err)

	branch := s.Engine.GetBranch("feature-1")

	// Lock the branch
	_, err = s.Engine.SetLocked([]engine.Branch{branch}, engine.LockReasonUser)
	require.NoError(t, err)

	// Verify locked
	impl := s.Engine.(interface{ Git() git.Runner })
	meta, err := impl.Git().ReadMetadata("feature-1")
	require.NoError(t, err)
	assert.Equal(t, git.LockReasonUser, meta.LockReason)

	// Unlock the branch
	_, err = s.Engine.SetLocked([]engine.Branch{branch}, engine.LockReasonNone)
	require.NoError(t, err)

	// Verify unlocked
	meta, err = impl.Git().ReadMetadata("feature-1")
	require.NoError(t, err)
	assert.Equal(t, git.LockReasonNone, meta.LockReason)
}

func TestSetFrozen_UnfreezeBranches(t *testing.T) {
	t.Parallel()
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create and track a branch
	s.CreateBranch("feature-1").
		Commit("feature 1 change")
	err := s.Engine.TrackBranch(context.Background(), "feature-1", "main")
	require.NoError(t, err)

	branch := s.Engine.GetBranch("feature-1")

	// Freeze the branch
	_, err = s.Engine.SetFrozen([]engine.Branch{branch}, true)
	require.NoError(t, err)

	// Verify frozen
	impl := s.Engine.(interface{ Git() git.Runner })
	localMeta, err := impl.Git().ReadLocalMetadata("feature-1")
	require.NoError(t, err)
	assert.True(t, localMeta.Frozen)

	// Unfreeze the branch
	_, err = s.Engine.SetFrozen([]engine.Branch{branch}, false)
	require.NoError(t, err)

	// Verify unfrozen
	localMeta, err = impl.Git().ReadLocalMetadata("feature-1")
	require.NoError(t, err)
	assert.False(t, localMeta.Frozen)
}

func TestIsConcurrentModificationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := engine.IsConcurrentModificationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
