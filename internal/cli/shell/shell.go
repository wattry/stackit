// Package shell provides shell integration for stackit.
package shell

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewShellCmd creates the shell integration command
func NewShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Output shell integration for auto-cd into worktrees",
		Long: `Output shell integration that enables automatic directory changes.

When you create a stack with --worktree/-w, stackit will automatically
change your shell's working directory to the new worktree.

Add this to your shell configuration file:

  # For zsh (~/.zshrc):
  eval "$(stackit shell zsh)"

  # For bash (~/.bashrc):
  eval "$(stackit shell bash)"

  # For fish (~/.config/fish/config.fish):
  stackit shell fish | source

This is separate from shell completions. You likely want both:

  eval "$(stackit completion zsh)"
  eval "$(stackit shell zsh)"
`,
	}

	cmd.AddCommand(newShellZshCmd())
	cmd.AddCommand(newShellBashCmd())
	cmd.AddCommand(newShellFishCmd())

	return cmd
}

func newShellZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Output zsh shell integration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), zshIntegration)
			return nil
		},
	}
}

func newShellBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Output bash shell integration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), bashIntegration)
			return nil
		},
	}
}

func newShellFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Output fish shell integration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), fishIntegration)
			return nil
		},
	}
}

const zshIntegration = `# stackit shell integration for zsh
# This wraps the stackit command to enable auto-cd into worktrees

__stackit_wrap() {
    local output exit_code cd_path rerun

    # Run the real stackit command and capture output
    # Set env var so stackit knows shell integration is available
    output=$(STACKIT_SHELL_INTEGRATION=1 command stackit "$@")
    exit_code=$?

    # Print output, filtering out directives
    echo "$output" | while IFS= read -r line; do
        if [[ "$line" == __STACKIT_CD__:* ]] || [[ "$line" == __STACKIT_RERUN__ ]]; then
            : # skip directives
        else
            echo "$line"
        fi
    done

    # Extract directives from output
    cd_path=$(echo "$output" | grep '^__STACKIT_CD__:' | head -1 | cut -d: -f2-)
    rerun=$(echo "$output" | grep -q '^__STACKIT_RERUN__$' && echo 1)

    # Change directory if path was found and exists
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
        # Re-run original command if requested
        if [[ -n "$rerun" ]]; then
            STACKIT_SHELL_INTEGRATION=1 command stackit "$@"
        fi
    fi

    return $exit_code
}

# Create wrapper function for stackit
stackit() {
    __stackit_wrap "$@"
}
`

const bashIntegration = `# stackit shell integration for bash
# This wraps the stackit command to enable auto-cd into worktrees

__stackit_wrap() {
    local output exit_code cd_path rerun

    # Run the real stackit command and capture output
    # Set env var so stackit knows shell integration is available
    output=$(STACKIT_SHELL_INTEGRATION=1 command stackit "$@")
    exit_code=$?

    # Print output, filtering out directives
    echo "$output" | while IFS= read -r line; do
        if [[ "$line" == __STACKIT_CD__:* ]] || [[ "$line" == __STACKIT_RERUN__ ]]; then
            : # skip directives
        else
            echo "$line"
        fi
    done

    # Extract directives from output
    cd_path=$(echo "$output" | grep '^__STACKIT_CD__:' | head -1 | cut -d: -f2-)
    rerun=$(echo "$output" | grep -q '^__STACKIT_RERUN__$' && echo 1)

    # Change directory if path was found and exists
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
        # Re-run original command if requested
        if [[ -n "$rerun" ]]; then
            STACKIT_SHELL_INTEGRATION=1 command stackit "$@"
        fi
    fi

    return $exit_code
}

# Create wrapper function for stackit
stackit() {
    __stackit_wrap "$@"
}
`

const fishIntegration = `# stackit shell integration for fish
# This wraps the stackit command to enable auto-cd into worktrees

function stackit --wraps=stackit --description 'stackit with auto-cd support'
    # Set env var so stackit knows shell integration is available
    set -l output (env STACKIT_SHELL_INTEGRATION=1 command stackit $argv)
    set -l exit_code $status

    # Print output, filtering out directives
    set -l cd_path ""
    set -l rerun 0
    for line in $output
        if string match -q '__STACKIT_CD__:*' -- $line
            set cd_path (string replace '__STACKIT_CD__:' '' -- $line)
        else if test "$line" = "__STACKIT_RERUN__"
            set rerun 1
        else
            echo $line
        end
    end

    # Change directory if path was found and exists
    if test -n "$cd_path" -a -d "$cd_path"
        cd $cd_path
        # Re-run original command if requested
        if test $rerun -eq 1
            env STACKIT_SHELL_INTEGRATION=1 command stackit $argv
        end
    end

    return $exit_code
end
`
