package git

// RebaseResult represents the result of a rebase operation
type RebaseResult int

const (
	// RebaseDone indicates the rebase was successful
	RebaseDone RebaseResult = iota
	// RebaseConflict indicates a conflict occurred during rebase
	RebaseConflict
)
