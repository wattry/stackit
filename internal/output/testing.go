package output

import (
	"bytes"
	"io"
	"os"
)

// TestOutput captures output for testing assertions.
type TestOutput struct {
	*ConsoleOutput
	buffer *bytes.Buffer
}

// NewTestOutput creates an output that captures all messages for testing.
func NewTestOutput() *TestOutput {
	buf := &bytes.Buffer{}
	return &TestOutput{
		ConsoleOutput: NewConsoleOutput(buf, true), // Enable debug for tests
		buffer:        buf,
	}
}

// String returns all captured output as a string.
func (t *TestOutput) String() string {
	return t.buffer.String()
}

// Bytes returns all captured output as bytes.
func (t *TestOutput) Bytes() []byte {
	return t.buffer.Bytes()
}

// Reset clears the captured output.
func (t *TestOutput) Reset() {
	t.buffer.Reset()
}

// NullOutput discards all output (for quiet tests).
type NullOutput struct{}

// NewNullOutput creates an output that discards everything.
func NewNullOutput() *NullOutput {
	return &NullOutput{}
}

// Info discards the message.
func (n *NullOutput) Info(_ string, _ ...any) {}

// Warn discards the message.
func (n *NullOutput) Warn(_ string, _ ...any) {}

// Error discards the message.
func (n *NullOutput) Error(_ string, _ ...any) {}

// Tip discards the message.
func (n *NullOutput) Tip(_ string, _ ...any) {}

// Success discards the message.
func (n *NullOutput) Success(_ string, _ ...any) {}

// Debug discards the message.
func (n *NullOutput) Debug(_ string, _ ...any) {}

// Print discards the content.
func (n *NullOutput) Print(_ string) {}

// Println discards the content.
func (n *NullOutput) Println(_ string) {}

// Newline does nothing.
func (n *NullOutput) Newline() {}

// DirectiveCD discards the directive.
func (n *NullOutput) DirectiveCD(_ string) {}

// SetQuiet does nothing.
func (n *NullOutput) SetQuiet(_ bool) {}

// IsQuiet always returns true.
func (n *NullOutput) IsQuiet() bool { return true }

var (
	// DefaultConsoleWriter is the writer used when no writer is specified.
	// This can be overridden in tests to capture output.
	DefaultConsoleWriter io.Writer = os.Stdout
)
