package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestRemoteMetadataSync(t *testing.T) {
	t.Run("detects and resolves metadata conflicts", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// 1. Create a branch with local metadata
		sh.CreateBranch("feature-a").
			CommitChange("file-a", "content-a").
			TrackBranch("feature-a", "main")

		eng := sh.Engine

		// Set local metadata: locked=false, scope="local-scope"
		branch := eng.GetBranch("feature-a")
		require.NoError(t, eng.SetLocked(branch, false))
		require.NoError(t, eng.SetScope(branch, engine.NewScope("local-scope")))

		// Verify local metadata
		require.False(t, eng.IsLocked(branch))
		require.Equal(t, "local-scope", eng.GetScope(branch).String())

		// 2. Simulate remote metadata with different values (locked=true, scope="remote-scope")
		// This simulates what would happen after `git fetch origin refs/stackit/metadata/*:refs/stackit/remote-metadata/*`
		remoteMeta := &engine.Meta{
			Locked: true,
			Scope:  strPtr("remote-scope"),
			LastModifiedBy: &engine.ModifiedBy{
				GitName:  "Remote User",
				GitEmail: "remote@example.com",
			},
		}
		createRemoteMetadataRef(t, sh, "feature-a", remoteMeta)

		// 3. Load remote metadata cache
		err := eng.LoadRemoteMetadataCache()
		require.NoError(t, err)

		// 4. Compute metadata diff
		diff, err := eng.ComputeMetadataDiff("feature-a")
		require.NoError(t, err)
		require.NotNil(t, diff, "expected diff to be non-nil")
		require.True(t, diff.HasConflict, "expected conflict to be detected")
		require.Len(t, diff.Differences, 2, "expected 2 field differences (locked and scope)")

		// Verify the diff contains the expected fields
		fieldNames := make(map[string]bool)
		for _, fd := range diff.Differences {
			fieldNames[fd.Field] = true
		}
		require.True(t, fieldNames["locked"], "expected 'locked' field in diff")
		require.True(t, fieldNames["scope"], "expected 'scope' field in diff")

		// 5. Accept remote metadata
		err = eng.AcceptRemoteMetadata("feature-a")
		require.NoError(t, err)

		// 6. Verify local metadata now matches remote
		require.True(t, eng.IsLocked(branch), "expected branch to be locked after accepting remote")
		require.Equal(t, "remote-scope", eng.GetScope(branch).String(), "expected scope to match remote after accepting")
	})

	t.Run("no conflict when local equals remote", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		sh.CreateBranch("feature-b").
			CommitChange("file-b", "content-b").
			TrackBranch("feature-b", "main")

		eng := sh.Engine

		// Set local metadata
		branch := eng.GetBranch("feature-b")
		require.NoError(t, eng.SetLocked(branch, true))
		require.NoError(t, eng.SetScope(branch, engine.NewScope("same-scope")))

		// Create identical remote metadata
		remoteMeta := &engine.Meta{
			Locked: true,
			Scope:  strPtr("same-scope"),
		}
		createRemoteMetadataRef(t, sh, "feature-b", remoteMeta)

		// Load remote cache
		err := eng.LoadRemoteMetadataCache()
		require.NoError(t, err)

		// Compute diff - should have no conflict
		diff, err := eng.ComputeMetadataDiff("feature-b")
		require.NoError(t, err)
		require.NotNil(t, diff)
		require.False(t, diff.HasConflict, "expected no conflict when local equals remote")
		require.Len(t, diff.Differences, 0, "expected no differences")
	})

	t.Run("detects orphaned local metadata", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		sh.CreateBranch("feature-c").
			CommitChange("file-c", "content-c").
			TrackBranch("feature-c", "main")

		eng := sh.Engine

		// Set local metadata and simulate it was previously synced
		branch := eng.GetBranch("feature-c")
		require.NoError(t, eng.SetLocked(branch, true))

		// Simulate that this metadata was synced from remote by setting LastModifiedBy
		// (which triggers LocalOnlyHash to be set)
		err := eng.SetLastModifiedBy("feature-c")
		require.NoError(t, err)

		// Load empty remote cache (simulating remote metadata was deleted)
		err = eng.LoadRemoteMetadataCache()
		require.NoError(t, err)

		// Find orphaned metadata
		orphaned, err := eng.FindOrphanedLocalMetadata()
		require.NoError(t, err)
		require.Len(t, orphaned, 1, "expected 1 orphaned metadata entry")
		require.Equal(t, "feature-c", orphaned[0].BranchName)
	})

	t.Run("HasLocalModifications detects changes since sync", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		sh.CreateBranch("feature-d").
			CommitChange("file-d", "content-d").
			TrackBranch("feature-d", "main")

		eng := sh.Engine
		branch := eng.GetBranch("feature-d")

		// Initial state: not modified (never synced)
		require.False(t, eng.HasLocalModifications("feature-d"))

		// Simulate sync by setting LastModifiedBy (this sets LocalOnlyHash)
		err := eng.SetLastModifiedBy("feature-d")
		require.NoError(t, err)

		// Still not modified (hash matches)
		require.False(t, eng.HasLocalModifications("feature-d"))

		// Now make a local change
		require.NoError(t, eng.SetLocked(branch, true))

		// Should now be detected as modified
		require.True(t, eng.HasLocalModifications("feature-d"))
	})
}

// createRemoteMetadataRef creates a ref at refs/stackit/remote-metadata/<branch> to simulate fetched remote metadata
func createRemoteMetadataRef(t *testing.T, sh *scenario.Scenario, branchName string, meta *engine.Meta) {
	t.Helper()

	// Serialize metadata to JSON
	data, err := json.Marshal(meta)
	require.NoError(t, err)

	// Write to a temp file so git hash-object can read it
	tmpFile := filepath.Join(sh.Scene.Dir, ".git", "tmp-meta-"+branchName)
	err = os.WriteFile(tmpFile, data, 0600)
	require.NoError(t, err)
	defer os.Remove(tmpFile)

	// Create the blob
	blobSha, err := sh.Scene.Repo.RunGitCommandAndGetOutput("hash-object", "-w", tmpFile)
	require.NoError(t, err)

	// Remove trailing newline
	blobSha = trimNewline(blobSha)

	// Create the ref
	refName := "refs/stackit/remote-metadata/" + branchName
	err = sh.Scene.Repo.RunGitCommand("update-ref", refName, blobSha)
	require.NoError(t, err)
}

func strPtr(s string) *string {
	return &s
}

func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}
