package config

// Git config keys for stackit configuration.
// All keys are prefixed with "stackit." to namespace them within git config.
const (
	// KeyTrunk is the primary trunk branch name.
	KeyTrunk = "stackit.trunk"
	// KeyTrunks stores additional trunk branches (multi-value).
	KeyTrunks = "stackit.trunks"
	// KeyBranchPattern is the pattern used for generating branch names.
	KeyBranchPattern = "stackit.branch.pattern"
	// KeySubmitFooter controls whether to include PR footer in submissions.
	KeySubmitFooter = "stackit.submit.footer"
	// KeyUndoDepth is the maximum number of undo snapshots to keep.
	KeyUndoDepth = "stackit.undo.depth"
	// KeyWorktreeBasePath is the base path for worktrees.
	KeyWorktreeBasePath = "stackit.worktree.basePath"
	// KeyWorktreeAutoClean controls automatic worktree cleanup during sync.
	KeyWorktreeAutoClean = "stackit.worktree.autoClean"
	// KeyMergeMethod is the preferred merge method (squash, merge, rebase).
	KeyMergeMethod = "stackit.merge.method"
	// KeyCICommand is the unified CI validation command.
	KeyCICommand = "stackit.ci.command"
	// KeyCITimeout is the CI command timeout in seconds.
	KeyCITimeout = "stackit.ci.timeout"
	// KeySplitHunkSelector is the hunk selector mode for split (tui or git).
	KeySplitHunkSelector = "stackit.split.hunkSelector"
	// KeyApprovedHooks stores approved post-worktree-create hooks (multi-value).
	KeyApprovedHooks = "stackit.hooks.approvedPostWorktreeCreate"
	// KeyMaxConcurrency is the maximum number of concurrent validation operations.
	KeyMaxConcurrency = "stackit.maxConcurrency"
	// KeyNavigationWhen controls when navigation is displayed (always/never/multiple).
	KeyNavigationWhen = "stackit.navigation.when"
	// KeyNavigationMarker is the custom marker symbol for the current branch.
	KeyNavigationMarker = "stackit.navigation.marker"
	// KeyNavigationLocation controls where navigation appears (body/comment).
	KeyNavigationLocation = "stackit.navigation.location"
	// KeyNavigationShowMerged controls whether to show merged branch history.
	KeyNavigationShowMerged = "stackit.navigation.showMerged"
)

// Default values for configuration.
const (
	// DefaultTrunk is the default trunk branch name.
	DefaultTrunk = "main"
	// DefaultSubmitFooter is whether to include PR footer by default.
	DefaultSubmitFooter = true
	// DefaultUndoDepth is the default number of undo snapshots to keep.
	DefaultUndoDepth = 10
	// DefaultWorktreeAutoClean is whether to auto-clean worktrees by default.
	DefaultWorktreeAutoClean = true
	// DefaultCITimeout is the default CI timeout in seconds (10 minutes).
	DefaultCITimeout = 600
	// DefaultSplitHunkSelector is the default hunk selector mode.
	DefaultSplitHunkSelector = "tui"
	// DefaultMaxConcurrency is the default max concurrent operations (0 = auto).
	DefaultMaxConcurrency = 0
	// DefaultNavigationWhen is the default navigation display mode.
	DefaultNavigationWhen = "multiple"
	// DefaultNavigationMarker is the default marker for the current branch.
	DefaultNavigationMarker = "👈"
	// DefaultNavigationLocation is the default location for navigation (PR body).
	DefaultNavigationLocation = "body"
	// DefaultNavigationShowMerged is whether to show merged history by default.
	DefaultNavigationShowMerged = true
)

// ValidMergeMethods contains the allowed merge method values.
var ValidMergeMethods = []string{"squash", "merge", "rebase"}

// ValidHunkSelectors contains the allowed hunk selector values.
var ValidHunkSelectors = []string{"tui", "git"}

// ValidNavigationWhen contains the allowed navigation when values.
var ValidNavigationWhen = []string{"always", "never", "multiple"}

// Navigation location constants.
const (
	NavigationLocationBody    = "body"
	NavigationLocationComment = "comment"
	NavigationLocationNone    = "none"
)

// ValidNavigationLocation contains the allowed navigation location values.
// "none" is an alias for disabling navigation (equivalent to when=never).
var ValidNavigationLocation = []string{NavigationLocationBody, NavigationLocationComment, NavigationLocationNone}
