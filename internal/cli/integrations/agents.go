// Package integrations provides commands for managing various integrations
package integrations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
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
	var includeAgentsMD bool

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

Subagent templates enable cost-effective delegation of tasks like commit message
and PR description generation to the faster Haiku model.

Use --include-agents-md to also add a stacking workflow block to your project's
CLAUDE.md or AGENTS.md file. This helps AI agents proactively think about
stacking during regular development work.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, _ := cmd.Flags().GetString("cwd")
			runner := git.NewRunner(nil)
			if cwd != "" {
				runner = git.NewRunnerWithPath(cwd, nil)
			}
			return runAgentInstall(runner, local, force, includeAgentsMD, version, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install files in current repository instead of globally")
	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")
	cmd.Flags().BoolVar(&includeAgentsMD, "include-agents-md", false, "Add stacking workflow block to CLAUDE.md or AGENTS.md")

	return cmd
}

func runAgentInstall(runner git.Runner, local, force, includeAgentsMD bool, version string, out io.Writer) error {
	var baseDir string
	var repoRoot string

	if local || includeAgentsMD {
		// Need repo root for local install or agents-md installation
		var err error
		repoRoot, err = runner.DiscoverRepoRoot()
		if err != nil && (local || includeAgentsMD) {
			return fmt.Errorf("not a git repository: %w", err)
		}
	}

	if local {
		// Local installation - install in current repo
		baseDir = repoRoot
	} else {
		// Global installation - install in home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = homeDir
	}

	// Check for existing installation and version
	if !force {
		if err := checkExistingInstallation(baseDir, version, out); err != nil {
			return err
		}
	}

	// Create .claude/skills/stackit directory
	skillDir := filepath.Join(baseDir, ".claude", "skills", "stackit")
	if err := os.MkdirAll(skillDir, 0750); err != nil {
		return fmt.Errorf("failed to create .claude/skills/stackit directory: %w", err)
	}

	// Write skill files
	skillFiles := map[string]string{
		"SKILL.md":     "agents/templates/skill/SKILL.md",
		"reference.md": "agents/templates/skill/reference.md",
	}

	for filename, templatePath := range skillFiles {
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		// Replace version placeholder with actual version
		contentStr := string(content)
		contentStr = strings.ReplaceAll(contentStr, "{{VERSION}}", version)

		filePath := filepath.Join(skillDir, filename)
		if err := os.WriteFile(filePath, []byte(contentStr), 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Create subdirectories for commands, workflows, and scripts
	commandRefDir := filepath.Join(skillDir, "commands")
	if err := os.MkdirAll(commandRefDir, 0750); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	workflowsDir := filepath.Join(skillDir, "workflows")
	if err := os.MkdirAll(workflowsDir, 0750); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	scriptsDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	subagentsDir := filepath.Join(skillDir, "subagents")
	if err := os.MkdirAll(subagentsDir, 0750); err != nil {
		return fmt.Errorf("failed to create subagents directory: %w", err)
	}

	// Write command reference files
	commandRefFiles := []string{
		"navigation.md",
		"branch.md",
		"stack.md",
		"recovery.md",
	}

	for _, filename := range commandRefFiles {
		templatePath := "agents/templates/skill/commands/" + filename
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		filePath := filepath.Join(commandRefDir, filename)
		if err := os.WriteFile(filePath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Write workflow files
	workflowFiles := []string{
		"absorb-conflict.md",
		"conflict-resolution.md",
		"fix-absorb.md",
		"stack-fold.md",
	}

	for _, filename := range workflowFiles {
		templatePath := "agents/templates/skill/workflows/" + filename
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		filePath := filepath.Join(workflowsDir, filename)
		if err := os.WriteFile(filePath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Write script files
	scriptFiles := []string{
		"analyze_stack.sh",
	}

	for _, filename := range scriptFiles {
		templatePath := "agents/templates/skill/scripts/" + filename
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		filePath := filepath.Join(scriptsDir, filename)
		if err := os.WriteFile(filePath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		// Make scripts executable (0700 = owner can read, write, execute)
		// #nosec G302 - Scripts need to be executable
		if err := os.Chmod(filePath, 0700); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", filename, err)
		}
	}

	// Write subagent files
	subagentFiles := []string{
		"commit-message.md",
		"review-triage.md",
	}

	for _, filename := range subagentFiles {
		templatePath := "agents/templates/subagents/" + filename
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		filePath := filepath.Join(subagentsDir, filename)
		if err := os.WriteFile(filePath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Create .claude/commands directory
	commandsDir := filepath.Join(baseDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0750); err != nil {
		return fmt.Errorf("failed to create .claude/commands directory: %w", err)
	}

	// Write slash command files
	commands := []string{
		"stack-absorb.md",
		"stack-create.md",
		"stack-extract.md",
		"stack-fix.md",
		"stack-fold.md",
		"stack-plan.md",
		"stack-restack.md",
		"stack-review.md",
		"stack-split.md",
		"stack-status.md",
		"stack-submit.md",
		"stack-sync.md",
		"stack-verify.md",
	}

	for _, filename := range commands {
		templatePath := "agents/templates/commands/" + filename
		content, err := agentTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		cmdPath := filepath.Join(commandsDir, filename)
		if err := os.WriteFile(cmdPath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Create .cursor/rules directory if it doesn't exist
	cursorRulesDir := filepath.Join(baseDir, ".cursor", "rules")
	if err := os.MkdirAll(cursorRulesDir, 0750); err != nil {
		return fmt.Errorf("failed to create .cursor/rules directory: %w", err)
	}

	// Write Cursor rules file
	cursorContent, err := agentTemplates.ReadFile("agents/templates/cursor/stackit.md")
	if err != nil {
		return fmt.Errorf("failed to read Cursor template: %w", err)
	}

	cursorRulesPath := filepath.Join(cursorRulesDir, "stackit.md")
	if err := os.WriteFile(cursorRulesPath, cursorContent, 0600); err != nil {
		return fmt.Errorf("failed to write Cursor rules file: %w", err)
	}

	// Install agents-md block if requested
	var agentsMDInstalled bool
	var agentsMDPath string
	if includeAgentsMD && repoRoot != "" {
		var blockErr error
		agentsMDInstalled, agentsMDPath, blockErr = installAgentsMDBlock(repoRoot, force)
		if blockErr != nil {
			return blockErr
		}
	}

	// Print success message
	installType := "globally"
	if local {
		installType = "locally"
	}

	_, _ = fmt.Fprintf(out, "✓ Installed agent files %s (version %s)\n\n", installType, version)
	_, _ = fmt.Fprintln(out, "Claude Code integration:")
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/SKILL.md\n", getDisplayPath(baseDir, local))
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/reference.md\n", getDisplayPath(baseDir, local))
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/commands/ (4 reference files)\n", getDisplayPath(baseDir, local))
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/workflows/ (4 workflow guides)\n", getDisplayPath(baseDir, local))
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/scripts/ (1 utility script)\n", getDisplayPath(baseDir, local))
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/skills/stackit/subagents/ (2 subagent templates)\n", getDisplayPath(baseDir, local))
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Slash commands:")
	_, _ = fmt.Fprintf(out, "✓ Created %s/.claude/commands/stack-*.md (%d commands)\n", getDisplayPath(baseDir, local), len(commands))
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Cursor integration:")
	_, _ = fmt.Fprintf(out, "✓ Created %s/.cursor/rules/stackit.md\n", getDisplayPath(baseDir, local))

	if agentsMDInstalled {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Agents MD block:")
		_, _ = fmt.Fprintf(out, "✓ Added stacking workflow block to %s\n", agentsMDPath)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Available Claude Code commands:")
	_, _ = fmt.Fprintln(out, "  /stack-absorb  - Intelligently absorb changes into commits")
	_, _ = fmt.Fprintln(out, "  /stack-create  - Create branch with auto-naming")
	_, _ = fmt.Fprintln(out, "  /stack-extract - Extract commits/files to independent branch")
	_, _ = fmt.Fprintln(out, "  /stack-fix     - Diagnose and fix stack issues")
	_, _ = fmt.Fprintln(out, "  /stack-fold    - Fold granular branches into parents")
	_, _ = fmt.Fprintln(out, "  /stack-plan    - Plan and create stack from uncommitted changes")
	_, _ = fmt.Fprintln(out, "  /stack-restack - Rebase all branches in stack")
	_, _ = fmt.Fprintln(out, "  /stack-review  - Apply PR review comments and mark resolved")
	_, _ = fmt.Fprintln(out, "  /stack-split   - Split changes between current and new child branch")
	_, _ = fmt.Fprintln(out, "  /stack-status  - View stack state and health")
	_, _ = fmt.Fprintln(out, "  /stack-submit  - Submit PRs with generated descriptions")
	_, _ = fmt.Fprintln(out, "  /stack-sync    - Sync with trunk and cleanup")
	_, _ = fmt.Fprintln(out, "  /stack-verify  - Verify stack health by running checks")

	if !local {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Tip: Use 'stackit agent install --local' to install files in a specific repository")
	}

	return nil
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

func getDisplayPath(_ string, local bool) string {
	if local {
		return "."
	}
	return "~"
}

const (
	agentsMDBlockStart = "<!-- stackit:start -->"
	agentsMDBlockEnd   = "<!-- stackit:end -->"
)

// installAgentsMDBlock adds the stacking workflow block to CLAUDE.md or AGENTS.md.
// Returns (installed, path, error) where installed indicates if the block was added/updated.
func installAgentsMDBlock(repoRoot string, force bool) (bool, string, error) {
	// Read the block template
	blockContent, err := agentTemplates.ReadFile("agents/templates/agents-block.md")
	if err != nil {
		return false, "", fmt.Errorf("failed to read agents-block template: %w", err)
	}

	// Find target file: prefer CLAUDE.md, fall back to AGENTS.md, create CLAUDE.md if neither
	targetFile := findAgentsFile(repoRoot)
	targetPath := filepath.Join(repoRoot, targetFile)

	// Read existing content (may not exist)
	existingContent, _ := os.ReadFile(targetPath)
	contentStr := string(existingContent)

	// Check for existing block
	if strings.Contains(contentStr, agentsMDBlockStart) {
		if !force {
			return false, targetFile, fmt.Errorf("stackit block already exists in %s, use --force to update", targetFile)
		}
		// Replace existing block
		contentStr = replaceAgentsMDBlock(contentStr, string(blockContent))
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

// findAgentsFile returns the appropriate agents file name for the repo.
// Returns CLAUDE.md if it exists, AGENTS.md if it exists, or CLAUDE.md as default.
func findAgentsFile(repoRoot string) string {
	claudePath := filepath.Join(repoRoot, "CLAUDE.md")
	if _, err := os.Stat(claudePath); err == nil {
		return "CLAUDE.md"
	}

	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err == nil {
		return "AGENTS.md"
	}

	return "CLAUDE.md"
}

// replaceAgentsMDBlock replaces the existing stackit block with new content.
func replaceAgentsMDBlock(content, newBlock string) string {
	startIdx := strings.Index(content, agentsMDBlockStart)
	endIdx := strings.Index(content, agentsMDBlockEnd)

	if startIdx == -1 || endIdx == -1 {
		return content
	}

	endIdx += len(agentsMDBlockEnd)

	// Preserve content before and after the block
	before := content[:startIdx]
	after := content[endIdx:]

	// Trim trailing newlines from before and leading from after to avoid double spacing
	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	result := before
	if len(before) > 0 {
		result += "\n\n"
	}
	result += newBlock
	if len(after) > 0 {
		result += "\n" + after
	}

	return result
}
