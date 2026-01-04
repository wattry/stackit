// Package doctor provides diagnostic functionality for checking stackit environment and repository health.
package doctor

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
)

// Options contains options for the doctor command
type Options struct {
	Fix   bool
	Trunk string // Trunk branch name from config
}

// Action runs diagnostic checks on the stackit environment and repository
func Action(ctx *app.Context, opts Options) error {
	out := ctx.Output
	eng := ctx.Engine

	if opts.Fix {
		out.Info("Running stackit doctor with --fix...")
	} else {
		out.Info("Running stackit doctor...")
	}
	out.Newline()

	var warnings []string
	var errors []string

	// Environment checks
	out.Info("Environment:")
	warnings, errors = checkEnvironment(ctx.Git(), out, warnings, errors)

	out.Newline()

	// Repository checks
	out.Info("Repository:")
	warnings, errors = checkRepository(ctx, out, warnings, errors, opts.Trunk)

	out.Newline()

	// Stack state checks
	out.Info("Stack State:")
	warnings, errors = checkStackState(eng, out, warnings, errors, opts.Fix)

	// Summary
	out.Newline()
	switch {
	case len(errors) > 0:
		out.Warn("Doctor found %d error(s) and %d warning(s).", len(errors), len(warnings))
		for _, err := range errors {
			out.Error("  %s", err)
		}
		for _, warn := range warnings {
			out.Warn("  %s", warn)
		}
		return fmt.Errorf("doctor found %d error(s)", len(errors))
	case len(warnings) > 0:
		if opts.Fix {
			out.Info("Doctor found %d warning(s), some of which may have been fixed.", len(warnings))
		} else {
			out.Info("Doctor found %d warning(s). Your stackit setup is mostly healthy.", len(warnings))
		}
		for _, warn := range warnings {
			out.Warn("  %s", warn)
		}
	default:
		out.Info("✅ All checks passed. Your stackit setup is healthy.")
	}

	return nil
}
