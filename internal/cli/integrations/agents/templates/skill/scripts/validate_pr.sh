#!/usr/bin/env bash
# Validate PR metadata quality before submission
# Part of stackit Claude Code skill

set -e

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m'

# Check if arguments provided
if [ $# -lt 2 ]; then
    echo "Usage: $0 <title> <body>"
    echo
    echo "Validates PR title and body for quality and completeness"
    exit 1
fi

TITLE="$1"
BODY="$2"

echo "=== PR Metadata Validation ==="
echo

# Track if validation passes
VALID=true

# Validate title
echo "Validating Title:"
echo "  \"$TITLE\""
echo

# Check title length
title_length=${#TITLE}
if [ $title_length -lt 10 ]; then
    echo -e "${RED}❌ Title too short (< 10 chars)${NC}"
    echo "   Titles should be descriptive and clear"
    VALID=false
elif [ $title_length -gt 100 ]; then
    echo -e "${YELLOW}⚠️  Title very long (> 100 chars)${NC}"
    echo "   Consider shortening for readability"
fi

# Check for conventional commit prefix (optional, just warn)
if echo "$TITLE" | grep -qE "^(feat|fix|docs|style|refactor|perf|test|chore|ci|build)(\(.+\))?:"; then
    echo -e "${GREEN}✓ Uses conventional commit format${NC}"
else
    echo -e "${YELLOW}ℹ️  Not using conventional commit format${NC}"
    echo "   Consider: feat: description, fix: description, etc."
fi

# Check for placeholder text
if echo "$TITLE" | grep -qiE "(todo|fixme|placeholder|test|example)"; then
    echo -e "${RED}❌ Title contains placeholder text${NC}"
    VALID=false
fi

echo

# Validate body
echo "Validating Body:"
echo

# Check body length
body_length=${#BODY}
if [ $body_length -lt 20 ]; then
    echo -e "${RED}❌ Body too short (< 20 chars)${NC}"
    echo "   Provide meaningful description of changes"
    VALID=false
elif [ $body_length -gt 5000 ]; then
    echo -e "${YELLOW}⚠️  Body very long (> 5000 chars)${NC}"
    echo "   Consider splitting into multiple PRs"
fi

# Check for placeholder text in body
if echo "$BODY" | grep -qiE "TODO|FIXME|placeholder|lorem ipsum|test test"; then
    echo -e "${RED}❌ Body contains placeholder text${NC}"
    VALID=false
fi

# Check for empty test plan placeholders
if echo "$BODY" | grep -qE "## Test Plan\s*$" || \
   echo "$BODY" | grep -qE "Test Plan.*\[\s*\]" || \
   echo "$BODY" | grep -qE "Test Plan.*TBD" || \
   echo "$BODY" | grep -qE "Test Plan.*TODO"; then
    echo -e "${YELLOW}⚠️  Test plan appears empty or placeholder${NC}"
    echo "   Add specific testing steps"
fi

# Check for summary section
if echo "$BODY" | grep -qiE "##? Summary"; then
    echo -e "${GREEN}✓ Contains summary section${NC}"
else
    echo -e "${YELLOW}ℹ️  No summary section found${NC}"
    echo "   Consider adding '## Summary' section"
fi

# Check for test plan section
if echo "$BODY" | grep -qiE "##? Test Plan"; then
    echo -e "${GREEN}✓ Contains test plan section${NC}"
else
    echo -e "${YELLOW}⚠️  No test plan section${NC}"
    echo "   Add '## Test Plan' with testing steps"
fi

# Check for bullet points or structure
if echo "$BODY" | grep -qE "^[\*\-]|^[0-9]+\."; then
    echo -e "${GREEN}✓ Uses structured formatting${NC}"
else
    echo -e "${YELLOW}ℹ️  No bullet points or numbered lists${NC}"
    echo "   Consider using lists for readability"
fi

# Check for code blocks (backticks)
if echo "$BODY" | grep -qE "\`\`\`"; then
    echo -e "${GREEN}✓ Contains code examples${NC}"
fi

echo
echo "=== Validation Result ==="

if [ "$VALID" = true ]; then
    echo -e "${GREEN}✓ PR metadata passes validation${NC}"
    echo
    echo "Ready to submit!"
    exit 0
else
    echo -e "${RED}✗ PR metadata needs improvement${NC}"
    echo
    echo "Please revise based on feedback above."
    exit 1
fi
