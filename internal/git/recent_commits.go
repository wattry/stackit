package git

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const trailerValueSeparator = "\x1e"

var prNumberSuffixRe = regexp.MustCompile(`\(#(\d+)\)\s*$`)

// RecentCommitKind describes the presentation type of a trunk commit.
type RecentCommitKind string

const (
	RecentCommitKindRegular    RecentCommitKind = "regular"
	RecentCommitKindStackMerge RecentCommitKind = "stack-merge"
)

// RecentCommit represents a commit from the git log with optional stack trailer metadata.
type RecentCommit struct {
	SHA            string
	Subject        string
	Author         string
	Date           time.Time
	PRNumber       int              // parsed from subject suffix "(#123)" if present
	Kind           RecentCommitKind // derived from trailer metadata
	StackSize      int              // from Stackit-Stack-Size trailer (0 if absent)
	StackPRNumbers []int            // from Stackit-PRs trailer
	StackScope     string           // from Stackit-Scope trailer (empty if absent)
}

// GetRecentCommits returns the most recent commits from a branch, including stack trailer metadata.
// Uses git log with a custom format string that includes trailer values.
func (r *runner) GetRecentCommits(branchName string, count int) ([]RecentCommit, error) {
	// Format: SHA\x1fsubject\x1fauthor\x1fdate\x1fstack-size\x1fprs\x1fscope
	// Use a non-empty trailer separator so duplicate keys remain parseable.
	format := "%H\x1f%s\x1f%an\x1f%aI\x1f%(trailers:key=Stackit-Stack-Size,valueonly,separator=%x1e)\x1f%(trailers:key=Stackit-PRs,valueonly,separator=%x1e)\x1f%(trailers:key=Stackit-Scope,valueonly,separator=%x1e)"

	output, err := r.runGitCommandInternal("log", fmt.Sprintf("-%d", count), "--format="+format, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits for %s: %w", branchName, err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var commits []RecentCommit
	for line := range strings.SplitSeq(output, "\n") {
		if line == "" {
			continue
		}

		fields := strings.SplitN(line, "\x1f", 7)
		if len(fields) < 4 {
			continue
		}

		commit := RecentCommit{
			SHA:     fields[0],
			Subject: fields[1],
			Author:  fields[2],
		}

		if date, parseErr := time.Parse(time.RFC3339, fields[3]); parseErr == nil {
			commit.Date = date
		}

		if len(fields) > 4 {
			commit.StackSize = parseStackSizeTrailer(fields[4])
		}

		if len(fields) > 5 {
			commit.StackPRNumbers = parseStackPRsTrailer(fields[5])
		}

		if len(fields) > 6 {
			commit.StackScope = parseStackScopeTrailer(fields[6])
		}

		commit.PRNumber = parsePRNumberFromSubject(commit.Subject)
		commit.Kind = deriveRecentCommitKind(commit)

		commits = append(commits, commit)
	}

	return commits, nil
}

func parseStackSizeTrailer(raw string) int {
	for value := range strings.SplitSeq(raw, trailerValueSeparator) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return 0
}

func parseStackPRsTrailer(raw string) []int {
	var prNumbers []int
	for value := range strings.SplitSeq(raw, trailerValueSeparator) {
		for part := range strings.SplitSeq(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.Atoi(part)
			if err != nil || slices.Contains(prNumbers, n) {
				continue
			}
			prNumbers = append(prNumbers, n)
		}
	}
	return prNumbers
}

func parseStackScopeTrailer(raw string) string {
	for value := range strings.SplitSeq(raw, trailerValueSeparator) {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parsePRNumberFromSubject(subject string) int {
	matches := prNumberSuffixRe.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return 0
	}
	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return n
}

func deriveRecentCommitKind(commit RecentCommit) RecentCommitKind {
	if commit.StackSize > 0 {
		return RecentCommitKindStackMerge
	}
	return RecentCommitKindRegular
}
