// Package output provides user-facing output and file logging.
package output

import (
	"fmt"
	"io"
	"os"
)

// Output handles user-facing console messages.
type Output interface {
	// Leveled messages with prefixes
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Tip(format string, args ...any)
	Success(format string, args ...any)
	Debug(format string, args ...any) // Only shown with debug flag

	// Raw output (no formatting)
	Print(content string)
	Println(content string)
	Newline()

	// Shell integration directives (always output, parsed by shell wrapper)
	DirectiveCD(path string) // Output __STACKIT_CD__:<path> for shell integration

	// Quiet mode for TUI coordination
	SetQuiet(quiet bool)
	IsQuiet() bool
}

// ConsoleOutput implements Output for terminal output.
type ConsoleOutput struct {
	writer    io.Writer
	debugMode bool
	quiet     bool
}

// NewConsoleOutput creates a new console output writer.
func NewConsoleOutput(writer io.Writer, debugMode bool) *ConsoleOutput {
	return &ConsoleOutput{
		writer:    writer,
		debugMode: debugMode,
		quiet:     false,
	}
}

// NewDefaultOutput creates a console output with stdout and debug from env.
func NewDefaultOutput() *ConsoleOutput {
	return NewConsoleOutput(os.Stdout, os.Getenv("DEBUG") != "")
}

// Info writes an info message.
func (c *ConsoleOutput) Info(format string, args ...any) {
	if c.quiet {
		return
	}
	c.printf(format, args...)
}

// Warn writes a warning message with emoji prefix.
func (c *ConsoleOutput) Warn(format string, args ...any) {
	if c.quiet {
		return
	}
	c.printf("⚠️  "+format, args...)
}

// Error writes an error message with emoji prefix.
func (c *ConsoleOutput) Error(format string, args ...any) {
	if c.quiet {
		return
	}
	c.printf("❌ "+format, args...)
}

// Tip writes a tip message with emoji prefix.
func (c *ConsoleOutput) Tip(format string, args ...any) {
	if c.quiet {
		return
	}
	c.printf("💡 "+format, args...)
}

// Success writes a success message with emoji prefix.
func (c *ConsoleOutput) Success(format string, args ...any) {
	if c.quiet {
		return
	}
	c.printf("✅ "+format, args...)
}

// Debug writes a debug message (only if debug mode is enabled).
func (c *ConsoleOutput) Debug(format string, args ...any) {
	if c.quiet || !c.debugMode {
		return
	}
	c.printf(format, args...)
}

// Print writes raw content without a newline.
func (c *ConsoleOutput) Print(content string) {
	if c.quiet {
		return
	}
	_, _ = fmt.Fprint(c.writer, content)
}

// Println writes content with a newline.
func (c *ConsoleOutput) Println(content string) {
	if c.quiet {
		return
	}
	_, _ = fmt.Fprintln(c.writer, content)
}

// Newline writes an empty line.
func (c *ConsoleOutput) Newline() {
	if c.quiet {
		return
	}
	_, _ = fmt.Fprintln(c.writer)
}

// DirectiveCD outputs a shell integration directive for changing directory.
// This is always output (even in quiet mode) as the shell wrapper needs to parse it.
func (c *ConsoleOutput) DirectiveCD(path string) {
	_, _ = fmt.Fprintf(c.writer, "__STACKIT_CD__:%s\n", path)
}

// SetQuiet enables or disables quiet mode.
func (c *ConsoleOutput) SetQuiet(quiet bool) {
	c.quiet = quiet
}

// IsQuiet returns whether quiet mode is enabled.
func (c *ConsoleOutput) IsQuiet() bool {
	return c.quiet
}

// printf is a helper that formats and prints a message with newline.
func (c *ConsoleOutput) printf(format string, args ...any) {
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	_, _ = fmt.Fprintln(c.writer, msg)
}
