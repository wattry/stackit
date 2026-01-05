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
    local exit_code cd_path rerun directive_file

    # Create temp file for directives (preserves TTY for interactive commands)
    directive_file=$(mktemp)

    # Run the real stackit command with full TTY access
    # Set env vars so stackit knows shell integration is available
    STACKIT_SHELL_INTEGRATION=1 STACKIT_DIRECTIVE_FILE="$directive_file" command stackit "$@"
    exit_code=$?

    # Read directives from temp file
    if [[ -f "$directive_file" ]]; then
        cd_path=$(grep '^__STACKIT_CD__:' "$directive_file" 2>/dev/null | head -1 | cut -d: -f2-)
        rerun=$(grep -q '^__STACKIT_RERUN__$' "$directive_file" 2>/dev/null && echo 1)
        rm -f "$directive_file"
    fi

    # Change directory if path was found and exists
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
        # Re-run original command if requested
        if [[ -n "$rerun" ]]; then
            STACKIT_SHELL_INTEGRATION=1 STACKIT_DIRECTIVE_FILE="" command stackit "$@"
        fi
    fi

    return $exit_code
}

# Create wrapper functions for stackit and st
stackit() { __stackit_wrap "$@"; }
st() { __stackit_wrap "$@"; }
`

const bashIntegration = `# stackit shell integration for bash
# This wraps the stackit command to enable auto-cd into worktrees

__stackit_wrap() {
    local exit_code cd_path rerun directive_file

    # Create temp file for directives (preserves TTY for interactive commands)
    directive_file=$(mktemp)

    # Run the real stackit command with full TTY access
    # Set env vars so stackit knows shell integration is available
    STACKIT_SHELL_INTEGRATION=1 STACKIT_DIRECTIVE_FILE="$directive_file" command stackit "$@"
    exit_code=$?

    # Read directives from temp file
    if [[ -f "$directive_file" ]]; then
        cd_path=$(grep '^__STACKIT_CD__:' "$directive_file" 2>/dev/null | head -1 | cut -d: -f2-)
        rerun=$(grep -q '^__STACKIT_RERUN__$' "$directive_file" 2>/dev/null && echo 1)
        rm -f "$directive_file"
    fi

    # Change directory if path was found and exists
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
        # Re-run original command if requested
        if [[ -n "$rerun" ]]; then
            STACKIT_SHELL_INTEGRATION=1 STACKIT_DIRECTIVE_FILE="" command stackit "$@"
        fi
    fi

    return $exit_code
}

# Create wrapper functions for stackit and st
stackit() { __stackit_wrap "$@"; }
st() { __stackit_wrap "$@"; }
`

const fishIntegration = `# stackit shell integration for fish
# This wraps the stackit command to enable auto-cd into worktrees

function __stackit_wrap --description 'stackit wrapper with auto-cd support'
    # Create temp file for directives (preserves TTY for interactive commands)
    set -l directive_file (mktemp)

    # Run the real stackit command with full TTY access
    # Set env vars so stackit knows shell integration is available
    env STACKIT_SHELL_INTEGRATION=1 STACKIT_DIRECTIVE_FILE="$directive_file" command stackit $argv
    set -l exit_code $status

    # Read directives from temp file
    set -l cd_path ""
    set -l rerun 0
    if test -f "$directive_file"
        set cd_path (grep '^__STACKIT_CD__:' "$directive_file" 2>/dev/null | head -1 | cut -d: -f2-)
        if grep -q '^__STACKIT_RERUN__$' "$directive_file" 2>/dev/null
            set rerun 1
        end
        rm -f "$directive_file"
    end

    # Change directory if path was found and exists
    if test -n "$cd_path" -a -d "$cd_path"
        cd $cd_path
        # Re-run original command if requested
        if test $rerun -eq 1
            env STACKIT_SHELL_INTEGRATION=1 STACKIT_DIRECTIVE_FILE="" command stackit $argv
        end
    end

    return $exit_code
end

# Create wrapper functions for stackit and st
function stackit --wraps=stackit --description 'stackit with auto-cd support'
    __stackit_wrap $argv
end

function st --wraps=stackit --description 'stackit with auto-cd support'
    __stackit_wrap $argv
end
`
