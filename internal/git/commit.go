package git

// CommitOptions contains options for creating a commit
type CommitOptions struct {
	Message     string
	Amend       bool
	NoEdit      bool
	Edit        bool
	Verbose     int
	ResetAuthor bool
	NoVerify    bool
}
