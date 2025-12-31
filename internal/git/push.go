package git

// PushOptions contains options for pushing a branch
type PushOptions struct {
	Force          bool
	ForceWithLease bool
	NoVerify       bool
}
