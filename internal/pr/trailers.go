package pr

import (
	"fmt"
	"strconv"
	"strings"
)

// Trailer key constants for stack metadata embedded in merge commits.
const (
	TrailerStackSize = "Stackit-Stack-Size"
	TrailerPRs       = "Stackit-PRs"
	TrailerScope     = "Stackit-Scope"
)

// StackTrailerInfo contains parsed stack metadata from git trailers.
type StackTrailerInfo struct {
	StackSize int
	PRNumbers []int
	Scope     string
}

// FormatStackTrailers builds a git trailer block for a consolidation merge commit.
// The trailers follow the standard git trailer format (key: value) and can be
// parsed by git log's %(trailers) format specifier.
func FormatStackTrailers(stackSize int, prNumbers []int, scope string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s: %d\n", TrailerStackSize, stackSize)

	if len(prNumbers) > 0 {
		prStrs := make([]string, len(prNumbers))
		for i, n := range prNumbers {
			prStrs[i] = strconv.Itoa(n)
		}
		fmt.Fprintf(&b, "%s: %s\n", TrailerPRs, strings.Join(prStrs, ","))
	}

	if scope != "" {
		fmt.Fprintf(&b, "%s: %s\n", TrailerScope, scope)
	}

	return b.String()
}

// ParseStackTrailers extracts stack trailer values from a commit message body.
// Returns nil if no stack trailers are found.
func ParseStackTrailers(body string) *StackTrailerInfo {
	info := &StackTrailerInfo{}
	found := false

	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)

		if val, ok := parseTrailer(line, TrailerStackSize); ok {
			if n, err := strconv.Atoi(val); err == nil {
				info.StackSize = n
				found = true
			}
		}

		if val, ok := parseTrailer(line, TrailerPRs); ok {
			info.PRNumbers = parsePRNumbers(val)
			if len(info.PRNumbers) > 0 {
				found = true
			}
		}

		if val, ok := parseTrailer(line, TrailerScope); ok {
			info.Scope = val
			found = true
		}
	}

	if !found {
		return nil
	}
	return info
}

// parseTrailer checks if a line matches "Key: value" and returns the value.
func parseTrailer(line, key string) (string, bool) {
	prefix := key + ":"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return strings.TrimSpace(line[len(prefix):]), true
}

// parsePRNumbers parses a comma-separated list of PR numbers.
func parsePRNumbers(s string) []int {
	var nums []int
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil {
			nums = append(nums, n)
		}
	}
	return nums
}
