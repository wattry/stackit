#!/usr/bin/env bash
# Analyze stack health and suggest actions
# Part of stackit Claude Code skill

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=== Stack Health Analysis ==="
echo

# Check if stackit is installed
if ! command -v stackit &> /dev/null; then
    echo -e "${RED}❌ stackit not found${NC}"
    echo "→ Install from: https://github.com/getstackit/stackit"
    exit 1
fi

# Check if in git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}❌ Not a git repository${NC}"
    exit 1
fi

# Check if stackit is initialized
if ! stackit log > /dev/null 2>&1; then
    echo -e "${RED}❌ Stackit not initialized in this repository${NC}"
    echo "→ Run: stackit init"
    exit 1
fi

echo -e "${BLUE}Current Stack:${NC}"
stackit log
echo

echo -e "${BLUE}Health Checks:${NC}"

# Check for uncommitted changes
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    echo -e "${YELLOW}⚠️  Uncommitted changes detected${NC}"
    echo "→ Run: git status"
    echo "→ Consider: git add . && stackit modify"
    echo
fi

# Check for branches without PRs
# Count branches marked with ● (no PR) in stackit log output
branches_no_pr=$(stackit log 2>/dev/null | grep -c "●" 2>/dev/null || echo "0")
if [ "$branches_no_pr" -gt 0 ]; then
    echo -e "${YELLOW}ℹ️  $branches_no_pr branch(es) without PRs${NC}"
    echo "→ Run: stackit submit --stack"
    echo
fi

# Check if sync needed (look for merged PRs)
if stackit log full 2>&1 | grep -qi "merged\|closed"; then
    echo -e "${YELLOW}ℹ️  Some PRs have been merged/closed${NC}"
    echo "→ Run: stackit sync --restack"
    echo
fi

# Check for rebase in progress
if [ -d "$(git rev-parse --git-dir)/rebase-merge" ] || [ -d "$(git rev-parse --git-dir)/rebase-apply" ]; then
    echo -e "${RED}⚠️  Rebase in progress${NC}"
    echo "→ Resolve conflicts and run: stackit continue"
    echo "→ Or abort with: stackit abort"
    echo
fi

# Get current branch
current_branch=$(git branch --show-current)

# Check if on trunk
trunk_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main")
if [ "$current_branch" = "$trunk_branch" ]; then
    echo -e "${YELLOW}ℹ️  Currently on trunk branch ($trunk_branch)${NC}"
    echo "→ Consider: stackit checkout (to switch to a feature branch)"
    echo
fi

# Run stackit doctor for additional diagnostics
echo -e "${BLUE}Running stackit doctor:${NC}"
if stackit doctor > /dev/null 2>&1; then
    echo -e "${GREEN}✓ No issues detected${NC}"
else
    echo -e "${YELLOW}⚠️  Stackit doctor found issues${NC}"
    echo "→ Run: stackit doctor (for details)"
    echo
fi

echo
echo "=== Analysis Complete ==="

# Suggest next action based on state
echo
echo -e "${BLUE}Suggested Next Steps:${NC}"

if [ "$branches_no_pr" -gt 0 ]; then
    echo "1. Submit branches: stackit submit --stack"
elif stackit log full 2>&1 | grep -qi "merged\|closed"; then
    echo "1. Sync with trunk: stackit sync --restack"
elif ! git diff-index --quiet HEAD -- 2>/dev/null; then
    echo "1. Commit changes: git add . && stackit modify"
else
    echo -e "${GREEN}✓ Stack is healthy! Ready for development.${NC}"
fi
