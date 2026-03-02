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
var mergeSubjectRe = regexp.MustCompile(`^Merge pull request #(\d+) from `)
var stackitTrailerRe = regexp.MustCompile(`^Stackit-[\w-]+:\s`)

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
// For merge commits ("Merge pull request #N from ..."), the subject is replaced with the
// first line of the body, which contains the actual PR title.
func (r *runner) GetRecentCommits(branchName string, count int) ([]RecentCommit, error) {
	// Format: SHA\x1fsubject\x1fauthor\x1fdate\x1fbody\x1fstack-size\x1fprs\x1fscope\x00
	// Records are separated by null bytes so multi-line bodies don't break parsing.
	format := "%H\x1f%s\x1f%an\x1f%aI\x1f%b\x1f%(trailers:key=Stackit-Stack-Size,valueonly,separator=%x1e)\x1f%(trailers:key=Stackit-PRs,valueonly,separator=%x1e)\x1f%(trailers:key=Stackit-Scope,valueonly,separator=%x1e)%x00"

	output, err := r.runGitCommandInternal("log", "--first-parent", fmt.Sprintf("-%d", count), "--format="+format, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits for %s: %w", branchName, err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var commits []RecentCommit
	for record := range strings.SplitSeq(output, "\x00") {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		fields := strings.SplitN(record, "\x1f", 8)
		if len(fields) < 4 {
			continue
		}

		subject := fields[1]
		body := ""
		if len(fields) > 4 {
			body = fields[4]
		}

		// For GitHub merge commits, the subject is "Merge pull request #N from ...".
		// The actual PR title is the first line of the body.
		subject = resolveSubject(subject, body)

		commit := RecentCommit{
			SHA:     fields[0],
			Subject: subject,
			Author:  fields[2],
		}

		if date, parseErr := time.Parse(time.RFC3339, fields[3]); parseErr == nil {
			commit.Date = date
		}

		if len(fields) > 5 {
			commit.StackSize = parseStackSizeTrailer(fields[5])
		}

		if len(fields) > 6 {
			commit.StackPRNumbers = parseStackPRsTrailer(fields[6])
		}

		if len(fields) > 7 {
			commit.StackScope = parseStackScopeTrailer(fields[7])
		}

		commit.PRNumber = parsePRNumberFromSubject(commit.Subject)
		if commit.PRNumber == 0 {
			commit.PRNumber = parseMergePRNumber(fields[1])
		}
		commit.Kind = deriveRecentCommitKind(commit)

		commits = append(commits, commit)
	}

	return commits, nil
}

// resolveSubject returns a human-readable subject for a commit.
// For GitHub merge commits ("Merge pull request #N from ..."), it extracts the
// actual PR title from the first line of the body.
func resolveSubject(subject, body string) string {
	if !mergeSubjectRe.MatchString(subject) {
		return subject
	}
	firstLine := firstNonEmptyLine(body)
	if firstLine != "" {
		return firstLine
	}
	return subject
}

// parseMergePRNumber extracts the PR number from a "Merge pull request #N from ..." subject.
func parseMergePRNumber(subject string) int {
	matches := mergeSubjectRe.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return 0
	}
	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return n
}

func firstNonEmptyLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip Stackit trailer lines (e.g. "Stackit-Stack-Size: 11")
		if stackitTrailerRe.MatchString(line) {
			continue
		}
		return line
	}
	return ""
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
