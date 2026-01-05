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
    local output exit_code cd_path

    # Run the real stackit command and capture output
    output=$(command stackit "$@")
    exit_code=$?

    # Print output, filtering out the __STACKIT_CD__ directive
    echo "$output" | while IFS= read -r line; do
        if [[ "$line" == __STACKIT_CD__:* ]]; then
            cd_path="${line#__STACKIT_CD__:}"
        else
            echo "$line"
        fi
    done

    # Extract cd path from output (in case the while loop runs in a subshell)
    cd_path=$(echo "$output" | grep '^__STACKIT_CD__:' | head -1 | cut -d: -f2-)

    # Change directory if path was found and exists
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
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
    local output exit_code cd_path

    # Run the real stackit command and capture output
    output=$(command stackit "$@")
    exit_code=$?

    # Print output, filtering out the __STACKIT_CD__ directive
    echo "$output" | while IFS= read -r line; do
        if [[ "$line" == __STACKIT_CD__:* ]]; then
            cd_path="${line#__STACKIT_CD__:}"
        else
            echo "$line"
        fi
    done

    # Extract cd path from output (in case the while loop runs in a subshell)
    cd_path=$(echo "$output" | grep '^__STACKIT_CD__:' | head -1 | cut -d: -f2-)

    # Change directory if path was found and exists
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
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
    set -l output (command stackit $argv)
    set -l exit_code $status

    # Print output, filtering out the __STACKIT_CD__ directive
    set -l cd_path ""
    for line in $output
        if string match -q '__STACKIT_CD__:*' -- $line
            set cd_path (string replace '__STACKIT_CD__:' '' -- $line)
        else
            echo $line
        end
    end

    # Change directory if path was found and exists
    if test -n "$cd_path" -a -d "$cd_path"
        cd $cd_path
    end

    return $exit_code
end
`
