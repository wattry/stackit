package integration

import (
	"bytes"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/cli"
)

// InProcessCLI provides in-process CLI execution for faster tests.
// Instead of spawning a new process for each command, it calls the CLI directly.
type InProcessCLI struct {
	mu sync.Mutex
}

// InProcessResult contains the result of an in-process CLI execution.
type InProcessResult struct {
	Output string
	Err    error
}

// NewInProcessCLI creates a new in-process CLI runner.
func NewInProcessCLI() *InProcessCLI {
	return &InProcessCLI{}
}

// Run executes a stackit command in-process.
// The workDir specifies the working directory for the command.
// Returns the combined stdout/stderr output and any error.
func (c *InProcessCLI) Run(workDir string, args ...string) InProcessResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create a new root command for each execution
	// The new runner architecture doesn't use global state, so no reset needed
	rootCmd := cli.NewRootCmd("test", "test", "test")

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Build args with --cwd for working directory and --no-interactive
	fullArgs := []string{"--cwd", workDir, "--no-interactive"}
	fullArgs = append(fullArgs, args...)
	rootCmd.SetArgs(fullArgs)

	// Execute
	err := rootCmd.Execute()

	return InProcessResult{
		Output: buf.String(),
		Err:    err,
	}
}

// RunString executes a stackit command from a single string (like "create feature -m 'test'").
func (c *InProcessCLI) RunString(workDir string, cmdStr string) InProcessResult {
	args := splitInProcessArgs(cmdStr)
	return c.Run(workDir, args...)
}

// splitInProcessArgs splits a command string into args, respecting quotes.
func splitInProcessArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case r == '"' || r == '\'':
			switch {
			case inQuote && r == quoteChar:
				inQuote = false
			case !inQuote:
				inQuote = true
				quoteChar = r
			default:
				current.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
