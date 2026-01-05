// Package output provides user-facing output and file logging.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
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
	DirectiveCD(path string)       // Output __STACKIT_CD__:<path> for shell integration
	DirectiveRerun(args ...string) // Output __STACKIT_RERUN__[:args] to run command after cd

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
// If STACKIT_DIRECTIVE_FILE is set, writes to that file instead of stdout.
func (c *ConsoleOutput) DirectiveCD(path string) {
	directive := fmt.Sprintf("__STACKIT_CD__:%s\n", path)
	writeDirective(c.writer, directive)
}

// DirectiveRerun outputs a shell integration directive to run a command after cd.
// If args are provided, runs "stackit <args...>". Otherwise re-runs the original command.
// This is always output (even in quiet mode) as the shell wrapper needs to parse it.
// If STACKIT_DIRECTIVE_FILE is set, writes to that file instead of stdout.
func (c *ConsoleOutput) DirectiveRerun(args ...string) {
	if len(args) == 0 {
		writeDirective(c.writer, "__STACKIT_RERUN__\n")
	} else {
		// Join args with spaces for shell to parse
		directive := fmt.Sprintf("__STACKIT_RERUN__:%s\n", joinArgs(args))
		writeDirective(c.writer, directive)
	}
}

// joinArgs joins arguments, quoting any that contain spaces.
func joinArgs(args []string) string {
	var result string
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		// Quote args containing spaces
		if strings.Contains(arg, " ") {
			result += fmt.Sprintf("%q", arg)
		} else {
			result += arg
		}
	}
	return result
}

// writeDirective writes a directive to the directive file if set, otherwise to the writer.
func writeDirective(fallback io.Writer, directive string) {
	if directiveFile := os.Getenv("STACKIT_DIRECTIVE_FILE"); directiveFile != "" {
		// Append to directive file for shell wrapper to read
		f, err := os.OpenFile(directiveFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = f.WriteString(directive)
			_ = f.Close()
			return
		}
		// Fall through to stdout on error
	}
	_, _ = fmt.Fprint(fallback, directive)
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
