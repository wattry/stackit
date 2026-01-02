package actions

import (
	"encoding/json"
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
)

// StackBranchInfo represents JSON-serializable info for a single branch in a stack
type StackBranchInfo struct {
	Name           string    `json:"name"`
	Parent         string    `json:"parent"`
	IsLocked       bool      `json:"is_locked"`
	IsFrozen       bool      `json:"is_frozen"`
	Scope          string    `json:"scope"`
	CommitMessages []string  `json:"commit_messages"`
	DiffStats      DiffStats `json:"diff_stats"`
}

// DiffStats represents summary diff information
type DiffStats struct {
	FilesChanged int `json:"files_changed"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
}

// StackInfoOptions contains options for the stack info logic
type StackInfoOptions struct {
	JSON bool
}

// StackInfoAction retrieves information about all branches in the current stack
func StackInfoAction(ctx *app.Context, opts StackInfoOptions) error {
	eng := ctx.Engine

	allBranches := eng.AllBranches()
	result := make([]StackBranchInfo, 0, len(allBranches))

	for _, branch := range allBranches {
		if branch.IsTrunk() {
			continue
		}

		info := StackBranchInfo{
			Name:     branch.GetName(),
			Parent:   branch.GetParentPrecondition(),
			IsLocked: branch.IsLocked(),
			IsFrozen: branch.IsFrozen(),
			Scope:    branch.GetScope().String(),
		}

		// Commit messages
		commits, err := branch.GetAllCommits(engine.CommitFormatReadable)
		if err == nil {
			info.CommitMessages = commits
		}

		// Diff stats
		added, deleted, err := branch.GetDiffStats()
		if err == nil {
			info.DiffStats.Additions = added
			info.DiffStats.Deletions = deleted
		}

		// Files changed
		parentName := branch.GetParentPrecondition()
		parentRev, err := eng.GetRevision(eng.GetBranch(parentName))
		if err == nil {
			branchRev, err := branch.GetRevision()
			if err == nil {
				files, err := eng.GetChangedFiles(ctx.Context, parentRev, branchRev)
				if err == nil {
					info.DiffStats.FilesChanged = len(files)
				}
			}
		}

		result = append(result, info)
	}

	if opts.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stack info to JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Non-JSON output not specified in plan, but let's provide a simple one or just fail
		// For now, let's just support JSON as requested.
		return fmt.Errorf("only --json is supported for --stack info")
	}

	return nil
}
