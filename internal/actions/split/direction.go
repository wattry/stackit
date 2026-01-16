package split

// Direction specifies where the new branch should be placed relative to the current branch
type Direction string

const (
	// DirectionBelow creates a new branch between current and parent (downstack).
	// This is the current/default behavior: the new branch becomes the parent of
	// the current branch.
	DirectionBelow Direction = "below"

	// DirectionAbove creates a new branch as a child of current (upstack).
	// The new branch becomes a child of the current branch, and any existing
	// children of current become children of the new branch.
	DirectionAbove Direction = "above"
)

// String returns the string representation of the direction
func (d Direction) String() string {
	return string(d)
}
