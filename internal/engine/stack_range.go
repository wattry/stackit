package engine

// StackRangeUpstack returns a range for children (upstack) traversal.
func StackRangeUpstack(includeCurrent bool) StackRange {
	return StackRange{
		RecursiveChildren: true,
		IncludeCurrent:    includeCurrent,
	}
}

// StackRangeDownstack returns a range for parents (downstack) traversal.
func StackRangeDownstack(includeCurrent bool) StackRange {
	return StackRange{
		RecursiveParents: true,
		IncludeCurrent:   includeCurrent,
	}
}

// StackRangeFull returns a range for full stack (parents + current + children).
func StackRangeFull() StackRange {
	return StackRange{
		RecursiveParents:  true,
		IncludeCurrent:    true,
		RecursiveChildren: true,
	}
}
