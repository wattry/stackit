package stack_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestSyncCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("sync when trunk is up to date", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run sync --no-restack (to isolate trunk sync output)
		cmd := exec.Command(binaryPath, "sync", "--no-restack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "sync command failed: %s", string(output))

		normalized := testhelpers.NormalizeOutput(string(output))
		require.Contains(t, normalized, "Pulling from remote", "should show pulling trunk")
		require.Contains(t, normalized, "is up to date", "should show trunk is up to date")
	})

	t.Run("sync with --restack when branches don't need restacking", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create a branch
			if err := s.Repo.CreateChange("branch1 change", "test1", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run sync with restack
		cmd := exec.Command(binaryPath, "sync", "--restack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "sync command failed: %s", string(output))

		normalized := testhelpers.NormalizeOutput(string(output))
		require.Contains(t, normalized, "Pulling from remote", "should show pulling trunk")
		require.Contains(t, normalized, "is up to date", "should show trunk is up to date")
		// Restack output - branch doesn't need restacking
		require.Contains(t, normalized, "branch1 does not need restacking", "should show branch1 doesn't need restacking")
	})

	t.Run("sync with --restack when branches need restacking", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "test1", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch2 on top of branch1
			if err := s.Repo.CreateChange("branch2 change", "test2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch2", "-m", "branch2 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Make a change to main so branches need restacking
		err := scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("main update", "main-file")
		require.NoError(t, err)

		// Switch to branch2 for sync
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)

		// Run sync with restack
		cmd := exec.Command(binaryPath, "sync", "--restack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "sync command failed: %s", string(output))

		normalized := testhelpers.NormalizeOutput(string(output))
		require.Contains(t, normalized, "Pulling from remote", "should show pulling trunk")
		// Since main was updated locally, remote is still at the old commit, so main is up to date
		require.Contains(t, normalized, "is up to date", "should show trunk is up to date")
		// Restack output - branches should be restacked (handler format uses -> revision)
		require.Contains(t, normalized, "Restacked branch1", "should show branch1 restacked")
		require.Contains(t, normalized, "Restacked branch2", "should show branch2 restacked")
	})

	t.Run("sync fails with uncommitted changes", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Create an uncommitted change
		err := scene.Repo.CreateChange("uncommitted", "uncommitted", false)
		require.NoError(t, err)

		// Run sync
		cmd := exec.Command(binaryPath, "sync")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "sync should fail with uncommitted changes")
		require.Contains(t, string(output), "uncommitted changes", "should mention uncommitted changes")
	})

	t.Run("sync with --no-restack shows tip", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create a branch
			if err := s.Repo.CreateChange("branch1 change", "test1", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run sync with --no-restack
		cmd := exec.Command(binaryPath, "sync", "--no-restack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "sync command failed: %s", string(output))

		normalized := testhelpers.NormalizeOutput(string(output))
		require.Contains(t, normalized, "--restack flag", "should show tip about --restack flag")
	})
}
