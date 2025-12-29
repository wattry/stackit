package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
)

var gitCommandAllowlist = []string{
	"add",
	"am",
	"apply",
	"archive",
	"bisect",
	"blame",
	"bundle",
	"cherry-pick",
	"clean",
	"clone",
	"commit",
	"diff",
	"difftool",
	"fetch",
	"format-patch",
	"fsck",
	"grep",
	// "merge" removed - stackit has its own merge command
	"mv",
	"notes",
	"pull",
	"push",
	"range-diff",
	"rebase",
	"reflog",
	"remote",
	"request-pull",
	"reset",
	"restore",
	"revert",
	"rm",
	"show",
	"send-email",
	"sparse-checkout",
	"stash",
	"status",
	"submodule",
	"switch",
	"tag",
}

var modifyingGitCommands = []string{
	"add",
	"am",
	"apply",
	"cherry-pick",
	"clean",
	"commit",
	"mv",
	"pull",
	"rebase",
	"reset",
	"restore",
	"revert",
	"rm",
	"stash",
}

// HandlePassthrough checks if the command should be passed through to git
// and executes it if so. Returns true if the command was handled (and the program should exit).
func HandlePassthrough(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Skip global flags to find the git command
	i := 1
	for i < len(args) {
		arg := args[i]

		// Handle flags with values
		if arg == "--cwd" {
			if i+1 < len(args) {
				_ = os.Chdir(args[i+1])
				i += 2
				continue
			}
		}

		// Handle boolean flags
		if slices.Contains([]string{"--debug", "--interactive", "--no-interactive", "--verify", "--no-verify", "--quiet", "-q"}, arg) {
			i++
			continue
		}

		// If it's a known git command, we've found our passthrough
		if slices.Contains(gitCommandAllowlist, arg) {
			command := arg
			gitArgs := args[i:]

			// Check if the command is modifying and the branch is locked
			if slices.Contains(modifyingGitCommands, command) {
				if locked, branch := isCurrentBranchLocked(); locked {
					fmt.Fprintf(os.Stderr, "Error: branch %s is locked. Use 'st unlock' to enable modifications.\n", branch)
					os.Exit(1)
				}
			}

			// Execute git command
			gitCmd := exec.Command("git", gitArgs...)
			gitCmd.Stdin = os.Stdin
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr

			// Print passthrough message
			fmt.Fprintf(os.Stderr, "\033[90mPassing command through to git...\033[0m\n")
			fmt.Fprintf(os.Stderr, "\033[90mRunning: \"git %s\"\033[0m\n\n", joinArgs(gitArgs))

			err := gitCmd.Run()
			if err != nil {
				var exitError *exec.ExitError
				if errors.As(err, &exitError) {
					os.Exit(exitError.ExitCode())
				}
				os.Exit(1)
			}
			os.Exit(0)
			return true
		}

		// Not a known git command and not a skipped flag, so stop
		return false
	}

	return false
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}

func isCurrentBranchLocked() (bool, string) {
	branch, err := git.RunGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return false, ""
	}
	branch = strings.TrimSpace(branch)

	refName := "refs/stackit/metadata/" + branch
	sha, err := git.RunGitCommand("rev-parse", "--verify", refName)
	if err != nil {
		return false, branch
	}

	content, err := git.RunGitCommand("cat-file", "-p", sha)
	if err != nil {
		return false, branch
	}

	var meta struct {
		Locked bool `json:"locked"`
	}
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return false, branch
	}

	return meta.Locked, branch
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "add [args...]",
		Short:              "git add passthrough",
		Long:               "arguments [args] (optional) git add arguments",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}
}

func newCherryPickCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "cherry-pick [args...]",
		Short:              "git cherry-pick passthrough",
		Long:               "arguments [args] (optional) git cherry-pick arguments",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}
}

func newRebaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "rebase [args...]",
		Short:              "git rebase passthrough",
		Long:               "arguments [args] (optional) git rebase arguments",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}
}

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "reset [args...]",
		Short:              "git reset passthrough",
		Long:               "arguments [args] (optional) git reset arguments",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}
}
