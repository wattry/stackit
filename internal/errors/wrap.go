// Package errors provides error wrapping helpers for consistent error messages.
package errors

import (
	"fmt"
)

// FailedTo wraps an error with a "failed to <action> <target>" message.
// Returns nil if err is nil, making it safe to use in error return statements.
//
// Example:
//
//	return errors.FailedTo("get", "branch revision", err)
//	// Returns: "failed to get branch revision: <underlying error>"
func FailedTo(action, target string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s %s: %w", action, target, err)
}

// While wraps an error with context about what action was being performed.
// Returns nil if err is nil, making it safe to use in error return statements.
//
// Example:
//
//	return errors.While("checking branch status", err)
//	// Returns: "checking branch status: <underlying error>"
func While(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}

// Op wraps an error with an operation name context.
// Returns nil if err is nil, making it safe to use in error return statements.
//
// Example:
//
//	return errors.Op("rebase", err)
//	// Returns: "rebase: <underlying error>"
func Op(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// Wrap wraps an error with arbitrary context.
// Returns nil if err is nil, making it safe to use in error return statements.
//
// Example:
//
//	return errors.Wrap(err, "processing branch %s", branchName)
//	// Returns: "processing branch feature: <underlying error>"
func Wrap(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
