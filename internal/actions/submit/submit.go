// Package submit provides functionality for submitting stacked branches as pull requests.
package submit

import (
	"errors"
	"fmt"
	"sync"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/utils"
)

// Options contains options for the submit command
type Options struct {
	Branch               string
	Stack                bool
	Force                bool
	DryRun               bool
	Confirm              bool
	UpdateOnly           bool
	Always               bool
	Restack              bool
	Draft                bool
	Publish              bool
	Edit                 bool
	EditTitle            bool
	EditDescription      bool
	NoEdit               bool
	NoEditTitle          bool
	NoEditDescription    bool
	Reviewers            string
	TeamReviewers        string
	MergeWhenReady       bool
	RerequestReview      bool
	View                 bool
	Web                  bool
	Comment              string
	TargetTrunk          string
	IgnoreOutOfSyncTrunk bool
	SubmitFooter         bool // Whether to include PR footer (from config)
}

// Info contains information about a branch to submit
type Info struct {
	BranchName string
	Head       string
	Base       string
	HeadSHA    string
	BaseSHA    string
	Action     string // "create" or "update"
	PRNumber   *int
	Metadata   *PRMetadata
}

// Action performs the submit operation with an event handler for progress feedback.
func Action(ctx *runtime.Context, opts Options, handler Handler) error {
	// Validate flags
	if opts.Draft && opts.Publish {
		return fmt.Errorf("can't use both --publish and --draft flags in one command")
	}

	// Get branches to submit
	branches, err := getBranchesToSubmit(ctx, opts)
	if err != nil {
		return err
	}
	if len(branches) == 0 {
		handler.OnEvent(CompletionEvent{Success: true, Message: "No branches to submit"})
		return nil
	}

	currentBranch := ctx.Engine.CurrentBranch()
	currentBranchName := ""
	if currentBranch != nil {
		currentBranchName = currentBranch.GetName()
	}

	// Populate remote SHAs early for accurate display
	if err := ctx.Engine.PopulateRemoteShas(); err != nil {
		ctx.Splog.Debug("Failed to populate remote SHAs: %v", err)
	}

	// Build tree structure for display
	branchObjs := make([]engine.Branch, len(branches))
	fixedMap := make(map[string]bool)
	scopeMap := make(map[string]string)

	for i, branchName := range branches {
		branch := ctx.Engine.GetBranch(branchName)
		branchObjs[i] = branch
		fixedMap[branchName] = branch.IsBranchUpToDate()
		scopeMap[branchName] = branch.GetScope().String()
	}

	stackTree := tree.NewStackTree(branchObjs, currentBranchName, ctx.Engine.Trunk().GetName())

	// Display the stack
	handler.OnEvent(StackDisplayEvent{
		Stack:    stackTree,
		FixedMap: fixedMap,
		ScopeMap: scopeMap,
	})

	// Restack if requested
	if opts.Restack {
		handler.OnEvent(RestackEvent{Started: true})
		// Convert []string to []engine.Branch for RestackBranches
		branchObjects := make([]engine.Branch, len(branches))
		for i, branchName := range branches {
			branchObjects[i] = ctx.Engine.GetBranch(branchName)
		}
		if err := actions.RestackBranches(ctx, branchObjects); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
		handler.OnEvent(RestackEvent{Completed: true})
	}

	// Validate and prepare branches
	handler.OnEvent(PreparingEvent{})

	if err := ValidateBranchesToSubmit(ctx, branches); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Prepare branches for submit (show planning phase with current indicator)
	submissionInfos, err := prepareBranchesForSubmit(ctx, branchObjs, opts, currentBranchName, handler)
	if err != nil {
		return fmt.Errorf("failed to prepare branches: %w", err)
	}

	// Check if we should abort
	if opts.DryRun {
		handler.OnEvent(CompletionEvent{Success: true, Message: "Dry run complete"})
		return nil
	}

	if len(submissionInfos) == 0 {
		handler.OnEvent(CompletionEvent{Success: true, Message: "All PRs up to date"})
		return nil
	}

	// Handle interactive confirmation
	if opts.Confirm {
		confirmed, err := handler.Confirm("Proceed with submit?", true)
		if err != nil {
			return fmt.Errorf("confirmation canceled: %w", err)
		}
		if !confirmed {
			handler.OnEvent(CompletionEvent{Success: false, Message: "Submit canceled"})
			return nil
		}
	}

	// Build branch info for submission start event
	branchInfos := make([]BranchInfo, len(submissionInfos))
	for i, info := range submissionInfos {
		branchInfos[i] = BranchInfo{
			Name:     info.BranchName,
			Action:   info.Action,
			PRNumber: info.PRNumber,
		}
	}

	// Start submission phase
	handler.OnEvent(SubmissionStartEvent{Branches: branchInfos})

	githubClient, err := getGitHubClient(ctx)
	if err != nil {
		return err
	}
	repoOwner, repoName := githubClient.GetOwnerRepo()

	remote := ctx.Engine.GetRemote()
	var wg sync.WaitGroup
	var submitErr error
	var errMu sync.Mutex

	for _, submissionInfo := range submissionInfos {
		wg.Add(1)
		go func(info Info) {
			defer wg.Done()

			handler.OnEvent(BranchProgressEvent{
				BranchName: info.BranchName,
				Status:     StatusSubmitting,
			})

			if err := pushBranchIfNeeded(ctx, info, opts, remote); err != nil {
				handler.OnEvent(BranchProgressEvent{
					BranchName: info.BranchName,
					Status:     StatusError,
					Error:      err,
				})
				errMu.Lock()
				if submitErr == nil {
					submitErr = err
				}
				errMu.Unlock()
				return
			}

			var prURL string
			const (
				actionCreate = "create"
				actionUpdate = "update"
			)
			var err error
			if info.Action == actionCreate {
				prURL, err = createPullRequestQuiet(ctx, info, repoOwner, repoName)
			} else {
				prURL, err = updatePullRequestQuiet(ctx, info, opts, repoOwner, repoName)
			}

			if err != nil {
				handler.OnEvent(BranchProgressEvent{
					BranchName: info.BranchName,
					Status:     StatusError,
					Error:      err,
				})
				errMu.Lock()
				if submitErr == nil {
					submitErr = err
				}
				errMu.Unlock()
				return
			}

			handler.OnEvent(BranchProgressEvent{
				BranchName: info.BranchName,
				Status:     StatusDone,
				URL:        prURL,
			})

			// Open in browser if requested
			if opts.View && prURL != "" {
				if err := utils.OpenBrowser(prURL); err != nil {
					ctx.Splog.Debug("Failed to open browser: %v", err)
				}
			}
		}(submissionInfo)
	}
	wg.Wait()

	if submitErr != nil {
		handler.OnEvent(CompletionEvent{Success: false, Message: "Submit failed"})
		return submitErr
	}

	// Update PR body footers silently
	if opts.SubmitFooter {
		actions.UpdateStackPRMetadata(ctx, branches, repoOwner, repoName)
	}

	// Push metadata refs for successfully submitted branches
	pushMetadataRefs(ctx, branchObjs)

	handler.OnEvent(CompletionEvent{Success: true, Message: "Submit complete"})
	return nil
}

// prepareBranchesForSubmit prepares submission info for each branch, emitting events via handler
func prepareBranchesForSubmit(ctx *runtime.Context, branches []engine.Branch, opts Options, currentBranch string, handler Handler) ([]Info, error) {
	submissionInfos := make([]Info, 0, len(branches))

	for _, branch := range branches {
		branchName := branch.GetName()
		status, err := branch.GetPRSubmissionStatus()
		if err != nil {
			return nil, err
		}

		action := status.Action
		prNumber := status.PRNumber
		prInfo := status.PRInfo

		isCurrent := branchName == currentBranch

		// Check if we should skip
		if opts.UpdateOnly && action == "create" {
			handler.OnEvent(BranchPlanEvent{
				BranchName: branchName,
				Action:     action,
				IsCurrent:  isCurrent,
				Skipped:    true,
				SkipReason: "skipped, no existing PR",
			})
			continue
		}

		needsUpdate := status.NeedsUpdate
		if action == "update" {
			// Check if draft status needs to change
			draftStatusNeedsChange := false
			if prInfo != nil {
				if opts.Draft && !prInfo.IsDraft() {
					draftStatusNeedsChange = true
				} else if opts.Publish && prInfo.IsDraft() {
					draftStatusNeedsChange = true
				}
			}

			needsUpdate = needsUpdate || opts.Edit || opts.Always || draftStatusNeedsChange

			if !needsUpdate && !opts.Draft && !opts.Publish {
				handler.OnEvent(BranchPlanEvent{
					BranchName: branchName,
					Action:     action,
					IsCurrent:  isCurrent,
					Skipped:    true,
					SkipReason: status.Reason,
				})
				continue
			}
		}

		// Prepare metadata
		metadataOpts := MetadataOptions{
			Edit:              opts.Edit && !opts.NoEdit,
			EditTitle:         opts.EditTitle && !opts.NoEditTitle,
			EditDescription:   opts.EditDescription && !opts.NoEditDescription,
			NoEdit:            opts.NoEdit,
			NoEditTitle:       opts.NoEditTitle,
			NoEditDescription: opts.NoEditDescription,
			Draft:             opts.Draft,
			Publish:           opts.Publish,
			Reviewers:         opts.Reviewers,
			ReviewersPrompt:   opts.Reviewers == "" && opts.Edit,
		}

		metadata, err := PreparePRMetadata(branch, metadataOpts, ctx.Engine, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare metadata for %s: %w", branchName, err)
		}

		// Get SHAs
		headSHA, _ := branch.GetRevision()
		parentBranchName := branch.GetParentPrecondition()
		parentBranch := ctx.Engine.GetBranch(parentBranchName)
		baseSHA, _ := parentBranch.GetRevision()

		submissionInfo := Info{
			BranchName: branchName,
			Head:       branchName,
			Base:       parentBranchName,
			HeadSHA:    headSHA,
			BaseSHA:    baseSHA,
			Action:     action,
			PRNumber:   prNumber,
			Metadata:   metadata,
		}

		handler.OnEvent(BranchPlanEvent{
			BranchName: branchName,
			Action:     action,
			IsCurrent:  isCurrent,
			Skipped:    false,
		})

		submissionInfos = append(submissionInfos, submissionInfo)
	}

	return submissionInfos, nil
}

// getBranchesToSubmit returns the list of branches to submit based on options
func getBranchesToSubmit(ctx *runtime.Context, opts Options) ([]string, error) {
	// Get branch scope
	branchName := opts.Branch
	if branchName == "" {
		currentBranch := ctx.Engine.CurrentBranch()
		if currentBranch == nil {
			return nil, fmt.Errorf("not on a branch and no branch specified")
		}
		branchName = currentBranch.GetName()
	}

	var allBranches []string
	if opts.Stack {
		// Include descendants and ancestors
		branch := ctx.Engine.GetBranch(branchName)
		stackBranches := branch.GetFullStack()
		allBranches = make([]string, len(stackBranches))
		for i, b := range stackBranches {
			allBranches[i] = b.GetName()
		}
	} else {
		// Just ancestors (including current branch)
		branch := ctx.Engine.GetBranch(branchName)
		downstackBranches := branch.GetRelativeStackDownstack()
		allBranches = make([]string, len(downstackBranches)+1)
		for i, b := range downstackBranches {
			allBranches[i] = b.GetName()
		}
		allBranches[len(downstackBranches)] = branchName
	}

	// Remove duplicates and trunk
	branches := []string{}
	branchSet := make(map[string]bool)
	for _, b := range allBranches {
		branchObj := ctx.Engine.GetBranch(b)
		if !branchObj.IsTrunk() && !branchSet[b] {
			branches = append(branches, b)
			branchSet[b] = true
		}
	}

	return branches, nil
}

// getGitHubClient returns the GitHub client from context
func getGitHubClient(ctx *runtime.Context) (github.Client, error) {
	if ctx.GitHubClient != nil {
		return ctx.GitHubClient, nil
	}
	return nil, fmt.Errorf("no GitHub client available - check your GITHUB_TOKEN")
}

// pushBranchIfNeeded pushes a branch to remote if needed
func pushBranchIfNeeded(ctx *runtime.Context, submissionInfo Info, opts Options, remote string) error {
	// Skip if dry run
	if opts.DryRun {
		return nil
	}

	forceWithLease := !opts.Force
	if err := ctx.Engine.PushBranch(ctx.Context, submissionInfo.BranchName, remote, git.PushOptions{
		Force:          opts.Force,
		ForceWithLease: forceWithLease,
		NoVerify:       !ctx.Verify,
	}); err != nil {
		if errors.Is(err, git.ErrStaleRemoteInfo) {
			return fmt.Errorf("force-with-lease push of %s failed due to external changes to the remote branch. If you are collaborating on this stack, try 'stackit sync' to pull in changes. Alternatively, use the --force option to bypass the stale info warning", submissionInfo.BranchName)
		}
		return fmt.Errorf("failed to push branch %s: %w", submissionInfo.BranchName, err)
	}
	return nil
}

// createPullRequestQuiet creates a new pull request without logging
func createPullRequestQuiet(ctx *runtime.Context, submissionInfo Info, repoOwner, repoName string) (string, error) {
	createOpts := github.CreatePROptions{
		Title:         submissionInfo.Metadata.Title,
		Body:          submissionInfo.Metadata.Body,
		Head:          submissionInfo.Head,
		Base:          submissionInfo.Base,
		Draft:         submissionInfo.Metadata.IsDraft,
		Reviewers:     submissionInfo.Metadata.Reviewers,
		TeamReviewers: submissionInfo.Metadata.TeamReviewers,
	}
	pr, err := ctx.GitHubClient.CreatePullRequest(ctx.Context, repoOwner, repoName, createOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create PR for %s: %w", submissionInfo.BranchName, err)
	}

	// Update PR info
	prNumber := pr.Number
	prURL := pr.HTMLURL
	branch := ctx.Engine.GetBranch(submissionInfo.BranchName)
	_ = ctx.Engine.UpsertPrInfo(branch, engine.NewPrInfo(
		&prNumber,
		submissionInfo.Metadata.Title,
		submissionInfo.Metadata.Body,
		"OPEN",
		submissionInfo.Base,
		prURL,
		submissionInfo.Metadata.IsDraft,
	))

	return prURL, nil
}

// updatePullRequestQuiet updates an existing pull request without logging
func updatePullRequestQuiet(ctx *runtime.Context, submissionInfo Info, opts Options, repoOwner, repoName string) (string, error) {
	// Check if base changed
	branch := ctx.Engine.GetBranch(submissionInfo.BranchName)
	prInfo, _ := branch.GetPrInfo()
	baseChanged := false
	if prInfo != nil && prInfo.Base() != submissionInfo.Base {
		baseChanged = true
	}

	updateOpts := github.UpdatePROptions{
		Title:           &submissionInfo.Metadata.Title,
		Body:            &submissionInfo.Metadata.Body,
		Reviewers:       submissionInfo.Metadata.Reviewers,
		TeamReviewers:   submissionInfo.Metadata.TeamReviewers,
		MergeWhenReady:  &opts.MergeWhenReady,
		RerequestReview: opts.RerequestReview,
	}

	// Only update draft status if it's explicitly set via flags
	if opts.Draft || opts.Publish {
		updateOpts.Draft = &submissionInfo.Metadata.IsDraft
	}

	// Before updating the base, check if there are commits between the new base and head
	// GitHub will reject the update if there are no commits between them
	baseToStore := submissionInfo.Base
	if baseChanged {
		baseUpdated := false
		// Only update base if there are commits between base and head
		if submissionInfo.BaseSHA != submissionInfo.HeadSHA {
			// Check if there are actually commits between base and head
			branch := ctx.Engine.GetBranch(submissionInfo.BranchName)
			commits, err := branch.GetAllCommits(engine.CommitFormatSHA)
			if err == nil && len(commits) > 0 {
				// There are commits, safe to update base
				updateOpts.Base = &submissionInfo.Base
				baseUpdated = true
			}
			// If no commits or error, skip base update to avoid GitHub 422 error
		}
		// If base SHA equals head SHA, skip base update (no commits between them)

		if !baseUpdated && prInfo != nil {
			// If we skipped the update, keep the existing base in our local cache
			// so it reflects what is actually on GitHub.
			baseToStore = prInfo.Base()
		}
	}

	if err := ctx.GitHubClient.UpdatePullRequest(ctx.Context, repoOwner, repoName, *submissionInfo.PRNumber, updateOpts); err != nil {
		return "", fmt.Errorf("failed to update PR for %s: %w", submissionInfo.BranchName, err)
	}

	// Get PR URL
	prInfo, _ = branch.GetPrInfo()
	var prURL string
	if prInfo != nil && prInfo.URL() != "" {
		prURL = prInfo.URL()
	} else {
		// Get from GitHub
		pr, err := ctx.GitHubClient.GetPullRequestByBranch(ctx.Context, repoOwner, repoName, submissionInfo.BranchName)
		if err == nil && pr != nil {
			prURL = pr.HTMLURL
		}
	}

	_ = ctx.Engine.UpsertPrInfo(branch, engine.NewPrInfo(
		submissionInfo.PRNumber,
		submissionInfo.Metadata.Title,
		submissionInfo.Metadata.Body,
		"OPEN",
		baseToStore,
		prURL,
		submissionInfo.Metadata.IsDraft,
	))

	return prURL, nil
}

// pushMetadataRefs pushes metadata refs for submitted branches to remote
func pushMetadataRefs(ctx *runtime.Context, branches []engine.Branch) {
	if len(branches) == 0 {
		return
	}

	// Update LastModifiedBy for each branch
	for _, branch := range branches {
		if err := ctx.Engine.SetLastModifiedBy(branch.GetName()); err != nil {
			ctx.Splog.Debug("Failed to set last modified by for %s: %v", branch.GetName(), err)
		}
	}

	// Check if remote sync is enabled; if not, run compatibility test first
	if !ctx.Engine.IsRemoteSyncEnabled() {
		if err := git.TestRemoteRefCompatibility(); err != nil {
			ctx.Splog.Debug("Remote metadata sync not supported: %v", err)
			return
		}
		ctx.Engine.SetRemoteSyncEnabled(true)
		// Configure refspec so future git fetch commands also fetch metadata
		if err := git.EnsureMetadataRefspecConfigured(); err != nil {
			ctx.Splog.Debug("Failed to configure metadata refspec: %v", err)
		}
	}

	// Extract branch names for git.PushMetadataRefs
	branchNames := make([]string, len(branches))
	for i, branch := range branches {
		branchNames[i] = branch.GetName()
	}

	// Push metadata refs
	if err := git.PushMetadataRefs(branchNames); err != nil {
		ctx.Splog.Debug("Failed to push metadata refs: %v", err)
	}
}
