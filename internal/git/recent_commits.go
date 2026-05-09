package git

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
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
// For merge commits ("Merge pull request #N from ..."), the subject is replaced with the
// first line of the body, which contains the actual PR title.
func (r *runner) GetRecentCommits(branchName string, count int) ([]RecentCommit, error) {
	if count <= 0 {
		return nil, nil
	}

	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}

	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

	currentHash, err := r.resolveRefHashInternal(repo, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve branch %s: %w", branchName, err)
	}

	commits := make([]RecentCommit, 0, count)
	for range count {
		commit, err := repo.CommitObject(currentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to load commit %s: %w", currentHash, err)
		}

		commits = append(commits, recentCommitFromGoGit(commit))
		if commit.NumParents() == 0 {
			break
		}
		currentHash = commit.ParentHashes[0]
		if currentHash == plumbing.ZeroHash {
			break
		}
	}

	return commits, nil
}

func recentCommitFromGoGit(commit *object.Commit) RecentCommit {
	subject, body := splitCommitMessage(commit.Message)
	resolvedSubject := resolveSubject(subject, body)

	result := RecentCommit{
		SHA:     commit.Hash.String(),
		Subject: resolvedSubject,
		Author:  commit.Author.Name,
		Date:    commit.Author.When,
	}

	result.StackSize = parseStackSizeTrailer(stackitTrailerValues(commit.Message, "Stackit-Stack-Size"))
	result.StackPRNumbers = parseStackPRsTrailer(stackitTrailerValues(commit.Message, "Stackit-PRs"))
	result.StackScope = parseStackScopeTrailer(stackitTrailerValues(commit.Message, "Stackit-Scope"))

	result.PRNumber = parsePRNumberFromSubject(result.Subject)
	if result.PRNumber == 0 {
		result.PRNumber = parseMergePRNumber(subject)
	}
	result.Kind = deriveRecentCommitKind(result)
	return result
}

func splitCommitMessage(message string) (subject, body string) {
	message = strings.TrimRight(message, "\n")
	if message == "" {
		return "", ""
	}
	subject, body, _ = strings.Cut(message, "\n")
	body = strings.TrimLeft(body, "\n")
	return strings.TrimSpace(subject), body
}

func stackitTrailerValues(message, key string) string {
	var values []string
	for line := range strings.SplitSeq(message, "\n") {
		trailerKey, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(trailerKey) != key {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			values = append(values, value)
		}
	}
	return strings.Join(values, trailerValueSeparator)
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
