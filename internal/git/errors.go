package git

import (
	"fmt"
)

// CommandError represents an error from a git command execution
type CommandError struct {
	Command string
	Args    []string
	Stdout  string
	Stderr  string
	Err     error
}

func (e *CommandError) Error() string {
	msg := fmt.Sprintf("git command failed: %s", e.Command)
	if len(e.Args) > 0 {
		msg += fmt.Sprintf(" %v", e.Args)
	}
	if e.Stderr != "" {
		msg += fmt.Sprintf("\nstderr: %s", e.Stderr)
	}
	if e.Stdout != "" {
		msg += fmt.Sprintf("\nstdout: %s", e.Stdout)
	}
	if e.Err != nil {
		msg += fmt.Sprintf("\n%v", e.Err)
	}
	return msg
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// NewCommandError creates a new CommandError
func NewCommandError(command string, args []string, stdout, stderr string, err error) *CommandError {
	return &CommandError{
		Command: command,
		Args:    args,
		Stdout:  stdout,
		Stderr:  stderr,
		Err:     err,
	}
}
