---
name: improve
description: Analyze changes in the current stack and provide actionable improvement suggestions using parallel multi-agent analysis. Reviews bugs, simplification opportunities, performance, test coverage, documentation, security, and UX. Keywords improve, code review, analysis, bugs, performance, security.
allowed-tools: Bash(git:*), Read, Grep, Glob, Task
---

# /improve - Code Improvement Analysis

Analyze changes in the current stack and provide actionable improvement suggestions across multiple dimensions.

## Usage

```
/improve [scope] [--agents=<list>]
```

**Scope options:**
- `stack` (default) - All changes from main to current branch
- `branch` - Only commits on current branch
- `staged` - Only staged changes

**Agent filter:**
- `--agents=all` (default) - Run all agents
- `--agents=bugs,perf` - Run specific agents only

## Execution Steps

### Step 1: Gather Context

First, collect information about the changes to analyze:

```bash
# Get the main/trunk branch name
git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main"

# Get current branch
git branch --show-current

# Get list of changed files
git diff --name-only main...HEAD

# Get the full diff for analysis
git diff main...HEAD

# Get diff stats
git diff --stat main...HEAD
```

Store this information for the agents:
- `TRUNK_BRANCH`: The main branch name
- `CURRENT_BRANCH`: Current branch name
- `CHANGED_FILES`: List of files changed
- `DIFF_CONTENT`: The actual diff
- `DIFF_STATS`: Summary statistics

### Step 2: Launch Analysis Agents

Launch ALL of the following agents in PARALLEL using the Task tool. Each agent receives the diff and changed files list.

**IMPORTANT:** You MUST launch all 7 agents simultaneously in a single message with multiple Task tool calls. Do not run them sequentially.

---

#### Agent 1: Bug Hunter (`bugs`)

```
You are analyzing code changes for potential bugs and edge cases.

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. Nil/null pointer dereferences
2. Missing error checks (especially in Go - never ignore errors)
3. Off-by-one errors in loops/slices
4. Race conditions in concurrent code
5. Resource leaks (unclosed files, connections, channels)
6. Unhandled edge cases (empty inputs, zero values, negative numbers)
7. Logic errors in conditionals
8. Missing input validation
9. Incorrect error wrapping (should use %w for wrapped errors)
10. Panic-inducing code paths

OUTPUT FORMAT (JSON):
{
  "agent": "bugs",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 142,
      "description": "Detailed explanation of the issue",
      "suggestion": "How to fix it",
      "effort": "LOW|MEDIUM|HIGH",
      "code_snippet": "relevant code if helpful"
    }
  ]
}

Only report ACTUAL issues found in the diff. Do not report speculative issues or issues in unchanged code.
If no issues found, return empty findings array.
```

---

#### Agent 2: Simplification (`simplify`)

```
You are analyzing code changes for simplification and refactoring opportunities.

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. Duplicated code that could be extracted
2. Complex conditionals that could be simplified
3. Deep nesting that could use early returns
4. Long functions that should be split
5. Dead code or unused variables
6. Unnecessary type conversions
7. Verbose patterns with simpler alternatives
8. Boolean parameters that should be typed constants
9. String literals that should be constants
10. Over-engineering (unnecessary abstractions)

PROJECT STYLE RULES:
- Early returns over deep nesting
- switch over if-else chains with 3+ conditions
- Use typed constants instead of boolean parameters
- Remove unused parameters entirely (don't use `_`)

OUTPUT FORMAT (JSON):
{
  "agent": "simplify",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 42,
      "description": "What could be simplified",
      "suggestion": "Proposed simplification",
      "effort": "LOW|MEDIUM|HIGH",
      "code_before": "current code",
      "code_after": "simplified code"
    }
  ]
}
```

---

#### Agent 3: Performance (`perf`)

```
You are analyzing code changes for performance issues.

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. Unnecessary allocations in hot paths
2. String concatenation in loops (use strings.Builder)
3. Unbounded slice growth (missing capacity hints)
4. N+1 query patterns
5. Missing caching opportunities
6. Blocking operations that could be async
7. Inefficient algorithms (O(n²) when O(n) is possible)
8. Repeated expensive computations
9. Large structs passed by value instead of pointer
10. Unnecessary regex compilation in loops

OUTPUT FORMAT (JSON):
{
  "agent": "perf",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 89,
      "description": "Performance issue explanation",
      "impact": "Expected impact on performance",
      "suggestion": "How to optimize",
      "effort": "LOW|MEDIUM|HIGH"
    }
  ]
}
```

---

#### Agent 4: Test Coverage (`tests`)

```
You are analyzing code changes for test coverage gaps.

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. New functions/methods without corresponding tests
2. New error paths without test coverage
3. Edge cases not covered by existing tests
4. Changed behavior that needs test updates
5. Complex conditionals with untested branches
6. Public API changes without integration tests
7. Missing table-driven test cases
8. Tests that don't use t.Parallel()

PROJECT TEST RULES:
- Always use t.Parallel() for parallel test execution
- Use require over assert for early failure
- Table-driven tests for multiple cases
- Integration tests use NewTestShellInProcess(t)

OUTPUT FORMAT (JSON):
{
  "agent": "tests",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 50,
      "description": "What needs test coverage",
      "test_suggestion": "Outline of test to add",
      "effort": "LOW|MEDIUM|HIGH"
    }
  ]
}
```

---

#### Agent 5: Documentation (`docs`)

```
You are analyzing code changes for documentation needs.

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. New public functions without doc comments
2. Changed behavior not reflected in existing docs
3. New CLI commands/flags without help text
4. Complex logic that needs explanatory comments
5. README updates needed for new features
6. Missing examples in help text
7. Outdated comments that don't match code
8. New configuration options without documentation

PROJECT DOC RULES:
- Comments explain "why" not "what"
- CLI Long descriptions should include examples
- Config changes go in docs/config.md
- TUI changes go in docs/tui.md

OUTPUT FORMAT (JSON):
{
  "agent": "docs",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 25,
      "description": "What documentation is needed",
      "suggestion": "Proposed documentation",
      "effort": "LOW|MEDIUM|HIGH"
    }
  ]
}
```

---

#### Agent 6: Security (`security`)

```
You are analyzing code changes for security issues (defensive review).

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. Command injection vulnerabilities (unescaped shell arguments)
2. Path traversal vulnerabilities
3. Hardcoded secrets or credentials
4. Insufficient input validation
5. TOCTOU (time-of-check-time-of-use) races
6. Unsafe deserialization
7. Missing authentication/authorization checks
8. Logging of sensitive data
9. Insecure temporary file creation
10. Unsafe use of user-controlled data

OUTPUT FORMAT (JSON):
{
  "agent": "security",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 78,
      "description": "Security issue explanation",
      "risk": "What could go wrong",
      "suggestion": "How to fix securely",
      "effort": "LOW|MEDIUM|HIGH"
    }
  ]
}
```

---

#### Agent 7: UX/Ergonomics (`ux`)

```
You are analyzing code changes for CLI user experience.

CHANGES TO ANALYZE:
- Branch: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
- Files: {CHANGED_FILES}

DIFF:
{DIFF_CONTENT}

LOOK FOR:
1. Unclear or unhelpful error messages
2. Missing progress indicators for long operations
3. Inconsistent command/flag naming
4. Missing confirmation prompts for destructive operations
5. Poor default values
6. Missing --help examples
7. Confusing output formatting
8. Missing color/formatting for important info
9. Unclear success/failure states
10. Missing keyboard shortcuts in TUI

OUTPUT FORMAT (JSON):
{
  "agent": "ux",
  "findings": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "title": "Brief description",
      "file": "path/to/file.go",
      "line": 112,
      "description": "UX issue explanation",
      "user_impact": "How this affects users",
      "suggestion": "Improved UX approach",
      "effort": "LOW|MEDIUM|HIGH"
    }
  ]
}
```

---

### Step 3: Collect and Parse Results

Wait for all agents to complete. Parse the JSON output from each agent.

### Step 4: Generate Report

Produce the final report in this format:

```
═══════════════════════════════════════════════════════════════════
 IMPROVE REPORT: {CURRENT_BRANCH} vs {TRUNK_BRANCH}
 Files changed: {count} | Lines added: {added} | Lines removed: {removed}
═══════════════════════════════════════════════════════════════════

[For each agent with findings, render a section:]

┌─ BUGS ({count} findings) ────────────────────────────────────────
│
│ [{SEVERITY}] {title}
│   {file}:{line}
│   {description}
│   Suggestion: {suggestion}
│   Effort: {effort}
│
│ [{SEVERITY}] {title}
│   ...
│
└──────────────────────────────────────────────────────────────────

[Repeat for: SIMPLIFY, PERF, TESTS, DOCS, SECURITY, UX]

═══════════════════════════════════════════════════════════════════
 SUMMARY
═══════════════════════════════════════════════════════════════════

 Priority Items (High impact + Low effort):
   1. {title} - {file}:{line}
   2. {title} - {file}:{line}
   ...

 Quick Wins (Low effort):
   • {title} ({agent})
   • {title} ({agent})
   ...

 By Category:
   Bugs:     {count} ({high} high, {med} medium, {low} low)
   Simplify: {count} ({high} high, {med} medium, {low} low)
   Perf:     {count} ({high} high, {med} medium, {low} low)
   Tests:    {count} ({high} high, {med} medium, {low} low)
   Docs:     {count} ({high} high, {med} medium, {low} low)
   Security: {count} ({high} high, {med} medium, {low} low)
   UX:       {count} ({high} high, {med} medium, {low} low)

 Total: {total} findings

═══════════════════════════════════════════════════════════════════
```

### Priority Ranking Logic

1. **Priority Items**: HIGH severity + LOW effort (fix these first!)
2. **Quick Wins**: Any severity + LOW effort
3. **Major Work**: HIGH severity + HIGH effort (plan for these)
4. **Nice to Have**: LOW severity + any effort

## Notes

- Only report issues actually present in the diff, not pre-existing issues
- Be specific with file paths and line numbers
- Provide actionable suggestions, not vague advice
- When in doubt about severity, be conservative (prefer MEDIUM over HIGH)
- Empty findings for an agent is fine - don't invent issues
