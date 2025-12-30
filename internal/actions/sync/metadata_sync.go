package sync

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
)

// syncRemoteMetadata fetches and processes remote metadata
func syncRemoteMetadata(ctx *runtime.Context, opts *Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Fetch remote metadata refs
	if err := git.FetchMetadataRefs(); err != nil {
		// Non-fatal: remote may not have metadata yet
		splog.Debug("No remote metadata to fetch: %v", err)
		return nil
	}

	// Configure refspec so future git fetch commands also fetch metadata
	if err := git.EnsureMetadataRefspecConfigured(); err != nil {
		splog.Debug("Failed to configure metadata refspec: %v", err)
	}

	// Load remote metadata into cache
	if err := eng.LoadRemoteMetadataCache(); err != nil {
		splog.Debug("Failed to load remote metadata cache: %v", err)
		return nil
	}

	// Compute diffs
	diffs, err := eng.ComputeAllMetadataDiffs()
	if err != nil {
		return fmt.Errorf("failed to compute metadata diffs: %w", err)
	}

	if len(diffs) == 0 {
		return nil // No conflicts
	}

	// Handle --dry-run flag
	if opts.DryRun {
		printMetadataDiffs(diffs, splog)
		return nil
	}

	// Prompt user for each conflicting branch
	for _, diff := range diffs {
		if err := promptAndResolveConflict(ctx, diff); err != nil {
			return err
		}
	}

	return nil
}

// printMetadataDiffs displays metadata differences in dry-run mode
func printMetadataDiffs(diffs []*engine.MetadataDiff, splog interface{ Info(string, ...interface{}) }) {
	splog.Info("\n=== Metadata changes (dry run) ===")
	for _, diff := range diffs {
		splog.Info("\nBranch: %s", style.ColorBranchName(diff.Branch, false))
		for _, fd := range diff.Differences {
			splog.Info("  %s: %v → %v", fd.Field, fd.LocalValue, fd.RemoteValue)
		}
	}
	splog.Info("\nRun without --dry-run to apply changes.")
}

// promptAndResolveConflict prompts the user to accept or reject remote metadata
func promptAndResolveConflict(ctx *runtime.Context, diff *engine.MetadataDiff) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Display field-level diff
	splog.Info("\nMetadata differs for branch '%s':", style.ColorBranchName(diff.Branch, false))
	for _, fd := range diff.Differences {
		splog.Info("  %s: %v (local) → %v (remote)", fd.Field, fd.LocalValue, fd.RemoteValue)
	}
	if diff.RemoteMeta.LastModifiedBy != nil {
		splog.Info("  Last modified by: %s <%s>",
			diff.RemoteMeta.LastModifiedBy.GitName,
			diff.RemoteMeta.LastModifiedBy.GitEmail)
	}

	// Prompt
	accept, err := promptYesNo("Accept remote metadata?")
	if err != nil {
		return err
	}

	if accept {
		return eng.AcceptRemoteMetadata(diff.Branch)
	}
	eng.RejectRemoteMetadata(diff.Branch)
	return nil
}

// promptYesNo prompts the user for a yes/no answer
func promptYesNo(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
