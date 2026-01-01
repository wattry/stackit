// Package agent provides commands for managing AI agent integration files
// for Cursor and Claude Code.
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
)

// NewAgentCmd creates the agent command
func NewAgentCmd(version string) *cobra.Command {
	// Set the template version to match the app version
	TemplateVersion = version

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agent integration files for Cursor and Claude Code",
		Long: `Manage agent integration files that help AI assistants use stackit effectively.

This command generates configuration files that enable AI agents (like Cursor and Claude Code)
to understand how to use stackit commands for managing stacked branches.`,
		SilenceUsage: true,
	}

	cmd.AddCommand(newAgentInstallCmd())

	return cmd
}

// newAgentInstallCmd creates the agent install command
func newAgentInstallCmd() *cobra.Command {
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
  - .claude/commands/ (Claude Code slash commands)

These files contain instructions for AI agents on how to use stackit commands
to manage stacked branches, create commits, submit PRs, and more.`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			runner := git.NewRunner()
			return runAgentInstall(runner, local, force)
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install files in current repository instead of globally")
	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")

	return cmd
}

func runAgentInstall(runner git.Runner, local, force bool) error {
	var baseDir string
	var err error

	if local {
		// Local installation - install in current repo
		repoRoot, err := runner.DiscoverRepoRoot()
		if err != nil {
			return fmt.Errorf("not a git repository: %w", err)
		}
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
		if err := checkExistingInstallation(baseDir); err != nil {
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
		"SKILL.md":     "templates/skill/SKILL.md",
		"reference.md": "templates/skill/reference.md",
	}

	for filename, templatePath := range skillFiles {
		content, err := skillTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		// Replace version placeholder with actual version
		contentStr := string(content)
		contentStr = strings.ReplaceAll(contentStr, "{{VERSION}}", TemplateVersion)

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

	// Write command reference files
	commandRefFiles := []string{
		"navigation.md",
		"branch.md",
		"stack.md",
		"recovery.md",
	}

	for _, filename := range commandRefFiles {
		templatePath := "templates/skill/commands/" + filename
		content, err := skillTemplates.ReadFile(templatePath)
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
		"fix-absorb.md",
		"conflict-resolution.md",
		"absorb-conflict.md",
	}

	for _, filename := range workflowFiles {
		templatePath := "templates/skill/workflows/" + filename
		content, err := skillTemplates.ReadFile(templatePath)
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
		"validate_pr.sh",
	}

	for _, filename := range scriptFiles {
		templatePath := "templates/skill/scripts/" + filename
		content, err := skillTemplates.ReadFile(templatePath)
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

	// Create .claude/commands directory
	commandsDir := filepath.Join(baseDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0750); err != nil {
		return fmt.Errorf("failed to create .claude/commands directory: %w", err)
	}

	// Write slash command files
	commands := []string{
		"stack-status.md",
		"stack-create.md",
		"stack-submit.md",
		"stack-absorb.md",
		"stack-fix.md",
		"stack-sync.md",
		"stack-restack.md",
	}

	for _, filename := range commands {
		templatePath := "templates/commands/" + filename
		content, err := commandTemplates.ReadFile(templatePath)
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
	cursorContent, err := cursorTemplates.ReadFile("templates/cursor/stackit.md")
	if err != nil {
		return fmt.Errorf("failed to read Cursor template: %w", err)
	}

	cursorRulesPath := filepath.Join(cursorRulesDir, "stackit.md")
	if err := os.WriteFile(cursorRulesPath, cursorContent, 0600); err != nil {
		return fmt.Errorf("failed to write Cursor rules file: %w", err)
	}

	// Print success message
	installType := "globally"
	if local {
		installType = "locally"
	}

	fmt.Printf("✓ Installed agent files %s (version %s)\n\n", installType, TemplateVersion)
	fmt.Println("Claude Code integration:")
	fmt.Printf("✓ Created %s/.claude/skills/stackit/SKILL.md\n", getDisplayPath(baseDir, local))
	fmt.Printf("✓ Created %s/.claude/skills/stackit/reference.md\n", getDisplayPath(baseDir, local))
	fmt.Printf("✓ Created %s/.claude/skills/stackit/commands/ (4 reference files)\n", getDisplayPath(baseDir, local))
	fmt.Printf("✓ Created %s/.claude/skills/stackit/workflows/ (3 workflow guides)\n", getDisplayPath(baseDir, local))
	fmt.Printf("✓ Created %s/.claude/skills/stackit/scripts/ (2 utility scripts)\n", getDisplayPath(baseDir, local))
	fmt.Println()
	fmt.Println("Slash commands:")
	fmt.Printf("✓ Created %s/.claude/commands/stack-*.md (7 commands)\n", getDisplayPath(baseDir, local))
	fmt.Println()
	fmt.Println("Cursor integration:")
	fmt.Printf("✓ Created %s/.cursor/rules/stackit.md\n", getDisplayPath(baseDir, local))

	fmt.Println()
	fmt.Println("Available Claude Code commands:")
	fmt.Println("  /stack-status  - View stack state and health")
	fmt.Println("  /stack-create  - Create branch with auto-naming")
	fmt.Println("  /stack-submit  - Submit PRs with generated descriptions")
	fmt.Println("  /stack-absorb  - Intelligently absorb changes into commits")
	fmt.Println("  /stack-fix     - Diagnose and fix stack issues")
	fmt.Println("  /stack-sync    - Sync with trunk and cleanup")
	fmt.Println("  /stack-restack - Rebase all branches in stack")

	if !local {
		fmt.Println()
		fmt.Println("Tip: Use 'stackit agent install --local' to install files in a specific repository")
	}

	return nil
}

func checkExistingInstallation(baseDir string) error {
	// Check if SKILL.md exists and has version info
	skillPath := filepath.Join(baseDir, ".claude", "skills", "stackit", "SKILL.md")
	if content, err := os.ReadFile(skillPath); err == nil {
		// File exists, check version
		existingVersion := extractVersion(string(content))
		if existingVersion != "" && existingVersion != TemplateVersion {
			fmt.Printf("Found existing installation (version %s)\n", existingVersion)
			fmt.Printf("New version available: %s\n", TemplateVersion)
			fmt.Println()
			fmt.Println("Run with --force to update")
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
