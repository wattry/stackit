// Package integrations provides commands for managing various integrations
package integrations

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	stackiterrors "stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
)

// NewAgentsCmd creates the agent command
func NewAgentsCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agent integration files for Claude Code and Codex",
		Long: `Manage agent integration files that help AI assistants use stackit effectively.

This command generates configuration files that enable AI agents (like Claude Code and Codex)
to understand how to use stackit commands for managing stacked branches.`,
		SilenceUsage: true,
	}

	cmd.AddCommand(newAgentInstallCmd(version))

	return cmd
}

// newAgentInstallCmd creates the agent install command
func newAgentInstallCmd(version string) *cobra.Command {
	var force bool
	var formats []string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install agent integration files",
		Long: `Install agent integration files for AI assistants.

By default, this command installs files in your home directory and prompts
you to select one or more skill folder formats.

This will create one or both:
  - ~/.claude/skills/stackit/ (Claude Code skill format)
  - ~/.codex/skills/stackit/ (Codex skill format)

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
			return runAgentInstall(runner, force, formats, version, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")
	cmd.Flags().StringSliceVar(&formats, "format", nil, "Skill format(s) to install (claude,codex). Repeat flag or use comma-separated values")

	return cmd
}

func runAgentInstall(runner git.Runner, force bool, formats []string, version string, out io.Writer) error {
	repoRoot, _ := runner.DiscoverRepoRoot()

	baseDir, err := resolveInstallBaseDir()
	if err != nil {
		return err
	}

	targets, err := selectInstallTargets(baseDir, formats)
	if err != nil {
		if errors.Is(err, stackiterrors.ErrCanceled) {
			return nil
		}
		return err
	}

	if err := confirmOverwriteIfNeeded(baseDir, targets, force, version, out); err != nil {
		if errors.Is(err, stackiterrors.ErrCanceled) {
			return nil
		}
		return err
	}

	for _, target := range targets {
		groups := buildAgentFileGroups(target)
		for _, g := range groups {
			if err := installFileGroup(baseDir, g, version); err != nil {
				return err
			}
		}

		switch target.format {
		case agentSkillFormatClaude:
			cleanupOldCommandFiles(baseDir, commandTemplateFiles)
			if err := installCommandSkills(baseDir, target.skillsBaseDir, commandTemplateFiles, renderClaudeSkillContent); err != nil {
				return err
			}
		case agentSkillFormatCodex:
			if err := installCommandSkills(baseDir, target.skillsBaseDir, commandTemplateFiles, renderCodexSkillContent); err != nil {
				return err
			}
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

	printSuccessMessage(out, targets, workflowBlockInstalled, workflowBlockPath, len(commandTemplateFiles))
	return nil
}

func printSuccessMessage(out io.Writer, targets []agentInstallTarget, workflowBlockInstalled bool, workflowBlockPath string, commandCount int) {
	_, _ = fmt.Fprintln(out, "✓ Installed agent files")

	for _, target := range targets {
		_, _ = fmt.Fprintf(out, "✓ Created %s\n", target.displayPath)
		_, _ = fmt.Fprintf(out, "✓ Created ~/%s/stack-*/ (%d skills)\n", target.skillsBaseDir, commandCount)
	}

	if workflowBlockInstalled {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Stacking workflow documentation:")
		_, _ = fmt.Fprintf(out, "✓ Added stacking workflow block to %s\n", workflowBlockPath)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Available Claude Code commands:")
	_, _ = fmt.Fprintln(out, "  /stack-absorb  - Intelligently absorb changes into commits")
	_, _ = fmt.Fprintln(out, "  /stack-create  - Create branch with auto-naming")
	_, _ = fmt.Fprintln(out, "  /stack-describe - Generate PR descriptions from git history")
	_, _ = fmt.Fprintln(out, "  /stack-extract - Extract commits/files to independent branch")
	_, _ = fmt.Fprintln(out, "  /stack-fix     - Diagnose and fix stack issues")
	_, _ = fmt.Fprintln(out, "  /stack-fold    - Fold granular branches into parents")
	_, _ = fmt.Fprintln(out, "  /stack-modify  - Amend current branch commit")
	_, _ = fmt.Fprintln(out, "  /stack-plan    - Plan and create stack from uncommitted changes")
	_, _ = fmt.Fprintln(out, "  /stack-resolve - Resolve rebase conflicts with AI assistance")
	_, _ = fmt.Fprintln(out, "  /stack-restack - Rebase all branches in stack")
	_, _ = fmt.Fprintln(out, "  /stack-review  - Apply PR review comments and mark resolved")
	_, _ = fmt.Fprintln(out, "  /stack-split   - Split changes between current and new child branch")
	_, _ = fmt.Fprintln(out, "  /stack-status  - View stack state and health")
	_, _ = fmt.Fprintln(out, "  /stack-submit  - Submit PRs with generated descriptions")
	_, _ = fmt.Fprintln(out, "  /stack-sync    - Sync with trunk and cleanup")
	_, _ = fmt.Fprintln(out, "  /stack-tidy    - Clean up fixup/WIP commits across the stack")
	_, _ = fmt.Fprintln(out, "  /stack-verify  - Verify stack health by running checks")
}
