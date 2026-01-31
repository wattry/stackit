// Package integrations provides commands for managing various integrations
package integrations

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	stackiterrors "stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// NewAgentsCmd creates the agent command
func NewAgentsCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agent integration files for Cursor and Claude Code",
		Long: `Manage agent integration files that help AI assistants use stackit effectively.

This command generates configuration files that enable AI agents (like Cursor and Claude Code)
to understand how to use stackit commands for managing stacked branches.`,
		SilenceUsage: true,
	}

	cmd.AddCommand(newAgentInstallCmd(version))

	return cmd
}

// newAgentInstallCmd creates the agent install command
func newAgentInstallCmd(version string) *cobra.Command {
	var local bool
	var force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install agent integration files",
		Long: `Install agent integration files for AI assistants.

By default, files are installed globally in ~/.claude/ and work across all repositories.
Use --local to install files in the current repository instead.

This will create:
  - .cursor/rules/stackit.md (for Cursor)
  - .claude/skills/stackit/ (Claude Code skill)
  - .claude/skills/stackit/subagents/ (Haiku subagent templates)
  - .claude/commands/ (Claude Code slash commands)

These files contain instructions for AI agents on how to use stackit commands
to manage stacked branches, create commits, submit PRs, and more.

When run in a git repository, you will be prompted to add a stacking workflow
block to your project's CLAUDE.md or AGENTS.md file.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, _ := cmd.Flags().GetString("cwd")
			runner := git.NewRunner(nil)
			if cwd != "" {
				runner = git.NewRunnerWithPath(cwd, nil)
			}
			return runAgentInstall(runner, local, force, version, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install files in current repository instead of globally")
	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")

	return cmd
}

// fileGroup defines a set of files to install from templates.
type fileGroup struct {
	templateDir string   // source directory in embedded fs (e.g., "agents/templates/skill/commands")
	destDir     string   // destination directory relative to baseDir
	files       []string // list of filenames
	executable  bool     // whether to make files executable
	replaceVer  bool     // whether to replace {{VERSION}} placeholder
}

func runAgentInstall(runner git.Runner, local, force bool, version string, out io.Writer) error {
	repoRoot, _ := runner.DiscoverRepoRoot()

	baseDir, err := resolveBaseDir(local, repoRoot)
	if err != nil {
		return err
	}

	if !force {
		if err := checkExistingInstallation(baseDir, version, out); err != nil {
			return err
		}
	}

	// Define all file groups to install
	skillDir := filepath.Join(".claude", "skills", "stackit")
	groups := []fileGroup{
		{
			templateDir: "agents/templates/skill",
			destDir:     skillDir,
			files:       []string{"SKILL.md", "reference.md"},
			replaceVer:  true,
		},
		{
			templateDir: "agents/templates/skill/commands",
			destDir:     filepath.Join(skillDir, "commands"),
			files:       []string{"navigation.md", "branch.md", "stack.md", "recovery.md"},
		},
		{
			templateDir: "agents/templates/skill/workflows",
			destDir:     filepath.Join(skillDir, "workflows"),
			files:       []string{"absorb-conflict.md", "conflict-resolution.md", "fix-absorb.md", "stack-fold.md"},
		},
		{
			templateDir: "agents/templates/skill/scripts",
			destDir:     filepath.Join(skillDir, "scripts"),
			files:       []string{"analyze_stack.sh"},
			executable:  true,
		},
		{
			templateDir: "agents/templates/subagents",
			destDir:     filepath.Join(skillDir, "subagents"),
			files:       []string{"commit-message.md", "review-triage.md"},
		},
		{
			templateDir: "agents/templates/commands",
			destDir:     filepath.Join(".claude", "commands"),
			files: []string{
				"stack-absorb.md", "stack-create.md", "stack-extract.md", "stack-fix.md",
				"stack-fold.md", "stack-plan.md", "stack-restack.md", "stack-review.md",
				"stack-split.md", "stack-status.md", "stack-submit.md", "stack-sync.md",
				"stack-verify.md",
			},
		},
		{
			templateDir: "agents/templates/cursor",
			destDir:     filepath.Join(".cursor", "rules"),
			files:       []string{"stackit.md"},
		},
	}

	// Install all file groups
	for _, g := range groups {
		if err := installFileGroup(baseDir, g, version); err != nil {
			return err
		}
	}

	// Install workflow block to CLAUDE.md or AGENTS.md if in a git repo
	var workflowBlockInstalled bool
	var workflowBlockPath string
	if repoRoot != "" {
		workflowBlockInstalled, workflowBlockPath, err = promptAndInstallWorkflowBlock(repoRoot, force)
		if err != nil {
			return err
		}
	}

	printSuccessMessage(out, local, workflowBlockInstalled, workflowBlockPath, len(groups[5].files))
	return nil
}

func resolveBaseDir(local bool, repoRoot string) (string, error) {
	if local {
		if repoRoot == "" {
			return "", fmt.Errorf("not a git repository: cannot use --local outside a git repository")
		}
		return repoRoot, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return homeDir, nil
}

func installFileGroup(baseDir string, g fileGroup, version string) error {
	destPath := filepath.Join(baseDir, g.destDir)
	if err := os.MkdirAll(destPath, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", g.destDir, err)
	}

	for _, filename := range g.files {
		templatePath := g.templateDir + "/" + filename
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		if g.replaceVer {
			content = []byte(strings.ReplaceAll(string(content), "{{VERSION}}", version))
		}

		filePath := filepath.Join(destPath, filename)
		if err := os.WriteFile(filePath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		if g.executable {
			// #nosec G302 - Scripts need to be executable
			if err := os.Chmod(filePath, 0700); err != nil {
				return fmt.Errorf("failed to make %s executable: %w", filename, err)
			}
		}
	}
	return nil
}

func printSuccessMessage(out io.Writer, local, workflowBlockInstalled bool, workflowBlockPath string, commandCount int) {
	displayPath := "~"
	installType := "globally"
	if local {
		displayPath = "."
		installType = "locally"
	}

	_, _ = fmt.Fprintf(out, "✓ Installed agent files %s\n\n", installType)
	_, _ = fmt.Fprintln(out, "Claude Code integration:")
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/ (skill + reference + commands + workflows + scripts + subagents)\n", displayPath)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Slash commands:")
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/commands/stack-*.md (%d commands)\n", displayPath, commandCount)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Cursor integration:")
	_, _ = fmt.Fprintf(out, "✓ Created %s/.cursor/rules/stackit.md\n", displayPath)

	if workflowBlockInstalled {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Stacking workflow documentation:")
		_, _ = fmt.Fprintf(out, "✓ Added stacking workflow block to %s\n", workflowBlockPath)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Available commands: /stack-absorb, /stack-create, /stack-extract, /stack-fix,")
	_, _ = fmt.Fprintln(out, "/stack-fold, /stack-plan, /stack-restack, /stack-review, /stack-split,")
	_, _ = fmt.Fprintln(out, "/stack-status, /stack-submit, /stack-sync, /stack-verify")

	if !local {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Tip: Use 'stackit agent install --local' to install in a specific repository")
	}
}

func checkExistingInstallation(baseDir, version string, out io.Writer) error {
	// Check if SKILL.md exists and has version info
	skillPath := filepath.Join(baseDir, ".claude", "skills", "stackit", "SKILL.md")
	if content, err := os.ReadFile(skillPath); err == nil {
		// File exists, check version
		existingVersion := extractVersion(string(content))
		if existingVersion != "" && existingVersion != version {
			_, _ = fmt.Fprintf(out, "Found existing installation (version %s)\n", existingVersion)
			_, _ = fmt.Fprintf(out, "New version available: %s\n", version)
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out, "Run with --force to update")
			return fmt.Errorf("existing installation found")
		}
	}

	return nil
}

func extractVersion(content string) string {
	// Look for version in frontmatter
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}

		if inFrontmatter && strings.HasPrefix(line, "version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

const (
	workflowBlockStart = "<!-- stackit:start -->"
	workflowBlockEnd   = "<!-- stackit:end -->"
)

// agentsFileInfo holds information about a potential agents file
type agentsFileInfo struct {
	name     string
	exists   bool
	hasBlock bool
	readErr  error // non-nil if file exists but couldn't be read (permission error, etc.)
	content  string
}

// discoverAgentsFiles checks for CLAUDE.md and AGENTS.md in the repo root.
func discoverAgentsFiles(repoRoot string) (claude, agents agentsFileInfo) {
	claude = checkAgentsFile(repoRoot, "CLAUDE.md")
	agents = checkAgentsFile(repoRoot, "AGENTS.md")
	return claude, agents
}

// checkAgentsFile checks if a specific agents file exists and its state.
func checkAgentsFile(repoRoot, filename string) agentsFileInfo {
	info := agentsFileInfo{name: filename}
	filePath := filepath.Join(repoRoot, filename)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist - that's fine
			return info
		}
		// File exists but we can't read it (permission error, etc.)
		info.exists = true
		info.readErr = err
		return info
	}

	info.exists = true
	info.content = string(content)
	info.hasBlock = strings.Contains(info.content, workflowBlockStart)
	return info
}

// promptAndInstallWorkflowBlock prompts the user and installs the workflow block.
// Returns (installed, path, error) where installed indicates if the block was added/updated.
func promptAndInstallWorkflowBlock(repoRoot string, force bool) (bool, string, error) {
	claude, agents := discoverAgentsFiles(repoRoot)

	// Check for read errors
	if claude.readErr != nil {
		return false, claude.name, fmt.Errorf("cannot read %s: %w", claude.name, claude.readErr)
	}
	if agents.readErr != nil {
		return false, agents.name, fmt.Errorf("cannot read %s: %w", agents.name, agents.readErr)
	}

	// Determine which file(s) are available
	var targetFile string
	switch {
	case claude.exists && agents.exists:
		// Both exist - prompt user to choose
		selected, err := tui.PromptSelect(
			"Both CLAUDE.md and AGENTS.md exist. Which file should receive the stacking workflow block?",
			[]tui.SelectOption{
				{Label: "CLAUDE.md", Value: "CLAUDE.md"},
				{Label: "AGENTS.md", Value: "AGENTS.md"},
				{Label: "Skip (don't add workflow block)", Value: "skip"},
			},
			0,
		)
		if errors.Is(err, stackiterrors.ErrCanceled) || errors.Is(err, tui.ErrInteractiveDisabled) {
			// User canceled or non-interactive mode - skip silently
			return false, "", nil
		}
		if err != nil {
			return false, "", fmt.Errorf("failed to prompt for file selection: %w", err)
		}
		if selected == "skip" {
			return false, "", nil
		}
		targetFile = selected

	case claude.exists:
		// Only CLAUDE.md exists - confirm
		confirmed, err := tui.PromptConfirm(
			fmt.Sprintf("Add stacking workflow block to %s?", claude.name),
			true,
		)
		if errors.Is(err, stackiterrors.ErrCanceled) || errors.Is(err, tui.ErrInteractiveDisabled) {
			return false, "", nil
		}
		if err != nil {
			return false, "", fmt.Errorf("failed to prompt for confirmation: %w", err)
		}
		if !confirmed {
			return false, "", nil
		}
		targetFile = claude.name

	case agents.exists:
		// Only AGENTS.md exists - confirm
		confirmed, err := tui.PromptConfirm(
			fmt.Sprintf("Add stacking workflow block to %s?", agents.name),
			true,
		)
		if errors.Is(err, stackiterrors.ErrCanceled) || errors.Is(err, tui.ErrInteractiveDisabled) {
			return false, "", nil
		}
		if err != nil {
			return false, "", fmt.Errorf("failed to prompt for confirmation: %w", err)
		}
		if !confirmed {
			return false, "", nil
		}
		targetFile = agents.name

	default:
		// Neither exists - ask if they want to create CLAUDE.md
		confirmed, err := tui.PromptConfirm(
			"No CLAUDE.md or AGENTS.md found. Create CLAUDE.md with stacking workflow block?",
			true,
		)
		if errors.Is(err, stackiterrors.ErrCanceled) || errors.Is(err, tui.ErrInteractiveDisabled) {
			return false, "", nil
		}
		if err != nil {
			return false, "", fmt.Errorf("failed to prompt for confirmation: %w", err)
		}
		if !confirmed {
			return false, "", nil
		}
		targetFile = "CLAUDE.md"
	}

	// Get the file info for the selected file
	var fileInfo agentsFileInfo
	if targetFile == "CLAUDE.md" {
		fileInfo = claude
	} else {
		fileInfo = agents
	}

	return installWorkflowBlock(repoRoot, targetFile, fileInfo, force)
}

// installWorkflowBlock installs the workflow block to the specified file.
func installWorkflowBlock(repoRoot, targetFile string, fileInfo agentsFileInfo, force bool) (bool, string, error) {
	// Read the block template
	blockContent, err := agentTemplates.ReadFile("agents/templates/agents-block.md")
	if err != nil {
		return false, "", fmt.Errorf("failed to read workflow block template: %w", err)
	}

	targetPath := filepath.Join(repoRoot, targetFile)
	contentStr := fileInfo.content

	// Check for existing block
	if fileInfo.hasBlock {
		if !force {
			return false, targetFile, fmt.Errorf("stackit block already exists in %s, use --force to update", targetFile)
		}
		// Replace existing block
		contentStr = replaceWorkflowBlock(contentStr, string(blockContent))
	} else {
		// Append block
		if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
			contentStr += "\n"
		}
		if len(contentStr) > 0 {
			contentStr += "\n"
		}
		contentStr += string(blockContent)
	}

	if err := os.WriteFile(targetPath, []byte(contentStr), 0600); err != nil {
		return false, targetFile, fmt.Errorf("failed to write %s: %w", targetFile, err)
	}

	return true, targetFile, nil
}

// replaceWorkflowBlock replaces the existing stackit block with new content.
func replaceWorkflowBlock(content, newBlock string) string {
	startIdx := strings.Index(content, workflowBlockStart)
	endIdx := strings.Index(content, workflowBlockEnd)

	// Handle missing or malformed markers
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return content
	}

	endIdx += len(workflowBlockEnd)

	// Preserve content before and after the block
	before := content[:startIdx]
	after := content[endIdx:]

	// Trim trailing newlines from before and leading from after to avoid double spacing
	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	var result strings.Builder
	result.WriteString(before)
	if len(before) > 0 {
		result.WriteString("\n\n")
	}
	result.WriteString(newBlock)
	if len(after) > 0 {
		result.WriteString("\n")
		result.WriteString(after)
	}

	return result.String()
}
