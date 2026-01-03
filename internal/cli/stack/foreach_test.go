package stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestForeachCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("output formatting", func(t *testing.T) {
		t.Parallel()
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 738ed75 (refactor: move tests around)
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)
		s.RunCli("init")
		s.RunCli("create", "branch1", "-m", "b1")
		s.RunCli("create", "branch2", "-m", "b2")
		s.RunGit("checkout", "branch1")
<<<<<<< HEAD

		// 1. Basic sequential output
		output, err := s.RunCliAndGetOutput("foreach", "echo", "hello")
		require.NoError(t, err)
		normalized := testhelpers.NormalizeOutput(output)
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch branch1 (current)...
hello
✓ Command succeeded on branch branch1 (current)
Running on branch branch2...
hello
✓ Command succeeded on branch branch2
Summary:
  ✓ branch1 (current)
    hello
  ✓ branch2
    hello
All branches completed successfully (2 total)
`), normalized)

		// 2. Parallel mode output
		output, err = s.RunCliAndGetOutput("foreach", "--parallel", "echo", "hi")
		require.NoError(t, err)
		normalized = testhelpers.NormalizeOutput(output)
		require.Equal(t, testhelpers.NormalizeOutput(`
Executing in parallel: ..
Summary:
  ✓ branch1 (current)
    hi
  ✓ branch2
    hi
All branches completed successfully (2 total)
`), normalized)

		// 3. Failure output
		output, err = s.RunCliAndGetOutput("foreach", "false")
		require.Error(t, err)
		normalized = testhelpers.NormalizeOutput(output)
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch branch1 (current)...
❌ ✗ Command failed on branch branch1 (current): exit status 1
Summary:
❌   ✗ branch1 (current)
❌     Error: exit status 1
Completed: 0 succeeded, 1 failed
Error: command failed on branch branch1: exit status 1
`), normalized)
=======
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			return runCliCommand(binaryPath, s.Dir, "create", "branch1", "-m", "branch1 change")
		})
=======
>>>>>>> 738ed75 (refactor: move tests around)

		// 1. Basic sequential output
		output, err := s.RunCliAndGetOutput("foreach", "echo", "hello")
		require.NoError(t, err)
		normalized := testhelpers.NormalizeOutput(output)
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch branch1 (current)...
hello
✓ Command succeeded on branch branch1 (current)
Running on branch branch2...
hello
✓ Command succeeded on branch branch2
Summary:
  ✓ branch1 (current)
    hello
  ✓ branch2
    hello
All branches completed successfully (2 total)
`), normalized)

<<<<<<< HEAD
		require.Equal(t, expected, normalized)
	})

	t.Run("foreach fail-fast", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2
			for _, b := range []string{"branch1", "branch2"} {
				if err := runCliCommand(binaryPath, s.Dir, "create", b, "-m", b+" change"); err != nil {
					return err
				}
			}
			return s.Repo.CheckoutBranch("branch1")
		})

		cmd := exec.Command(binaryPath, "foreach", "false")
		cmd.Dir = scene.Dir
		outputBytes, _ := cmd.CombinedOutput()
		output := string(outputBytes)
		normalized := testhelpers.NormalizeOutput(output)

		expected := testhelpers.NormalizeOutput(`
Running on branch branch1 (current)...
❌ ✗ Command failed on branch branch1 (current): exit status 1

Summary:
❌   ✗ branch1 (current)
❌     Error: exit status 1

Completed: 0 succeeded, 1 failed
Error: command failed on branch branch1: exit status 1
`)

		require.Equal(t, expected, normalized)
>>>>>>> 2188396 (refactor: foreach tests)
	})

	t.Run("scope flags", func(t *testing.T) {
		t.Parallel()
<<<<<<< HEAD
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)
		s.RunCli("init")
		s.RunCli("create", "b1")
		s.RunCli("create", "b2")
		s.RunCli("checkout", "b1")

		// Verify --stack includes everything
		output, _ := s.RunCliAndGetOutput("foreach", "--stack", "echo", "hi")
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch b1 (current)...
hi
✓ Command succeeded on branch b1 (current)
Running on branch b2...
hi
✓ Command succeeded on branch b2
Summary:
  ✓ b1 (current)
    hi
  ✓ b2
    hi
All branches completed successfully (2 total)
`), testhelpers.NormalizeOutput(output))

		// Verify default (upstack) only includes b1 and b2
		output, _ = s.RunCliAndGetOutput("foreach", "echo", "hi")
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch b1 (current)...
hi
✓ Command succeeded on branch b1 (current)
Running on branch b2...
hi
✓ Command succeeded on branch b2
Summary:
  ✓ b1 (current)
    hi
  ✓ b2
    hi
All branches completed successfully (2 total)
`), testhelpers.NormalizeOutput(output))

		// Verify --downstack from b1 only includes b1
		output, _ = s.RunCliAndGetOutput("foreach", "--downstack", "echo", "hi")
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch b1 (current)...
hi
✓ Command succeeded on branch b1 (current)
Summary:
  ✓ b1 (current)
    hi
All branches completed successfully (1 total)
`), testhelpers.NormalizeOutput(output))
=======
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2
			for _, b := range []string{"branch1", "branch2"} {
				if err := runCliCommand(binaryPath, s.Dir, "create", b, "-m", b+" change"); err != nil {
					return err
				}
			}
			return s.Repo.CheckoutBranch("branch1")
		})

		output := runCliCommandSuccess(t, binaryPath, scene.Dir, "foreach", "--no-fail-fast", "false")
		normalized := testhelpers.NormalizeOutput(output)

		expected := testhelpers.NormalizeOutput(`
Running on branch branch1 (current)...
❌ ✗ Command failed on branch branch1 (current): exit status 1

Running on branch branch2...
❌ ✗ Command failed on branch branch2: exit status 1

Summary:
❌   ✗ branch1 (current)
❌     Error: exit status 1
❌   ✗ branch2
❌     Error: exit status 1

Completed: 0 succeeded, 2 failed
`)

		require.Equal(t, expected, normalized)
	})

	t.Run("foreach on trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1
			if err := runCliCommand(binaryPath, s.Dir, "create", "branch1", "-m", "branch1 change"); err != nil {
				return err
			}
			// Checkout main
			return s.Repo.CheckoutBranch("main")
		})

		output := runCliCommandSuccess(t, binaryPath, scene.Dir, "foreach", "echo", "hello")
		normalized := testhelpers.NormalizeOutput(output)

		expected := testhelpers.NormalizeOutput(`
Running on branch branch1...
hello
✓ Command succeeded on branch branch1

Summary:
  ✓ branch1
    hello

All branches completed successfully (1 total)
`)

		require.Equal(t, expected, normalized)
	})

	t.Run("foreach --parallel", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2 -> branch3
			for _, b := range []string{"branch1", "branch2", "branch3"} {
				if err := runCliCommand(binaryPath, s.Dir, "create", b, "-m", b+" change"); err != nil {
					return err
				}
			}
			return s.Repo.CheckoutBranch("branch1")
		})

		output := runCliCommandSuccess(t, binaryPath, scene.Dir, "foreach", "--parallel", "echo", "hi")
		normalized := testhelpers.NormalizeOutput(output)

		// In parallel mode, we show dots for progress.
		expected := testhelpers.NormalizeOutput(`
Executing in parallel: ...
Summary:
  ✓ branch1 (current)
    hi
  ✓ branch2
    hi
  ✓ branch3
    hi

All branches completed successfully (3 total)
`)

		require.Equal(t, expected, normalized)
	})

	t.Run("foreach --parallel --jobs", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			for _, b := range []string{"branch1", "branch2"} {
				if err := runCliCommand(binaryPath, s.Dir, "create", b, "-m", b+" change"); err != nil {
					return err
				}
			}
			return s.Repo.CheckoutBranch("branch1")
		})

		output := runCliCommandSuccess(t, binaryPath, scene.Dir, "foreach", "--parallel", "--jobs", "1", "echo", "hi")
		normalized := testhelpers.NormalizeOutput(output)

		expected := testhelpers.NormalizeOutput(`
=======
		// 2. Parallel mode output
		output, err = s.RunCliAndGetOutput("foreach", "--parallel", "echo", "hi")
		require.NoError(t, err)
		normalized = testhelpers.NormalizeOutput(output)
		require.Equal(t, testhelpers.NormalizeOutput(`
>>>>>>> 738ed75 (refactor: move tests around)
Executing in parallel: ..
Summary:
  ✓ branch1 (current)
    hi
  ✓ branch2
    hi
All branches completed successfully (2 total)
`), normalized)

<<<<<<< HEAD
		require.Equal(t, expected, normalized)
>>>>>>> 2188396 (refactor: foreach tests)
=======
		// 3. Failure output
		output, err = s.RunCliAndGetOutput("foreach", "false")
		require.Error(t, err)
		normalized = testhelpers.NormalizeOutput(output)
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch branch1 (current)...
❌ ✗ Command failed on branch branch1 (current): exit status 1
Summary:
❌   ✗ branch1 (current)
❌     Error: exit status 1
Completed: 0 succeeded, 1 failed
Error: command failed on branch branch1: exit status 1
`), normalized)
	})

	t.Run("scope flags", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)
		s.RunCli("init")
		s.RunCli("create", "b1")
		s.RunCli("create", "b2")
		s.RunCli("checkout", "b1")

		// Verify --stack includes everything
		output, _ := s.RunCliAndGetOutput("foreach", "--stack", "echo", "hi")
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch b1 (current)...
hi
✓ Command succeeded on branch b1 (current)
Running on branch b2...
hi
✓ Command succeeded on branch b2
Summary:
  ✓ b1 (current)
    hi
  ✓ b2
    hi
All branches completed successfully (2 total)
`), testhelpers.NormalizeOutput(output))

		// Verify default (upstack) only includes b1 and b2
		output, _ = s.RunCliAndGetOutput("foreach", "echo", "hi")
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch b1 (current)...
hi
✓ Command succeeded on branch b1 (current)
Running on branch b2...
hi
✓ Command succeeded on branch b2
Summary:
  ✓ b1 (current)
    hi
  ✓ b2
    hi
All branches completed successfully (2 total)
`), testhelpers.NormalizeOutput(output))

		// Verify --downstack from b1 only includes b1
		output, _ = s.RunCliAndGetOutput("foreach", "--downstack", "echo", "hi")
		require.Equal(t, testhelpers.NormalizeOutput(`
Running on branch b1 (current)...
hi
✓ Command succeeded on branch b1 (current)
Summary:
  ✓ b1 (current)
    hi
All branches completed successfully (1 total)
`), testhelpers.NormalizeOutput(output))
>>>>>>> 738ed75 (refactor: move tests around)
	})
}
