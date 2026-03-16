package integrations

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// fileGroup defines a set of files to install from templates.
type fileGroup struct {
	templateDir string   // source directory in embedded fs (e.g., "agents/templates/skill/commands")
	destDir     string   // destination directory relative to baseDir
	files       []string // list of filenames
	executable  bool     // whether to make files executable
	replaceVer  bool     // whether to replace {{VERSION}} placeholder
}

type agentSkillFormat string

const (
	agentSkillFormatClaude agentSkillFormat = "claude"
	agentSkillFormatCodex  agentSkillFormat = "codex"
)

type agentInstallTarget struct {
	format            agentSkillFormat
	skillDir          string
	skillsBaseDir     string // base directory for individual skills (e.g., ".claude/skills" or ".codex/skills")
	displayPath       string
	includeAgentsMeta bool
}

type existingSkillInstallation struct {
	target  agentInstallTarget
	path    string
	version string
}

var (
	skillRootFiles = []string{"SKILL.md", "reference.md"}

	skillCommandFiles = []string{"navigation.md", "branch.md", "stack.md", "recovery.md"}

	skillWorkflowFiles = []string{"absorb-conflict.md", "conflict-resolution.md", "fix-absorb.md", "stack-fold.md"}

	skillScriptFiles = []string{"analyze_stack.sh"}

	subagentFiles = []string{"commit-message.md", "review-triage.md"}

	commandTemplateFiles = []string{
		"stack-absorb.md", "stack-create.md", "stack-describe.md", "stack-extract.md",
		"stack-fix.md", "stack-fold.md", "stack-modify.md", "stack-plan.md", "stack-resolve.md",
		"stack-restack.md", "stack-review.md", "stack-split.md", "stack-status.md",
		"stack-submit.md", "stack-sync.md", "stack-tidy.md", "stack-verify.md",
	}
)

func installTargetForFormat(format agentSkillFormat) agentInstallTarget {
	switch format {
	case agentSkillFormatCodex:
		return agentInstallTarget{
			format:            agentSkillFormatCodex,
			skillDir:          filepath.Join(".codex", "skills", "stackit"),
			skillsBaseDir:     filepath.Join(".codex", "skills"),
			displayPath:       "~/.codex/skills/stackit",
			includeAgentsMeta: true,
		}
	default:
		return agentInstallTarget{
			format:        agentSkillFormatClaude,
			skillDir:      filepath.Join(".claude", "skills", "stackit"),
			skillsBaseDir: filepath.Join(".claude", "skills"),
			displayPath:   "~/.claude/skills/stackit",
		}
	}
}

func buildAgentFileGroups(target agentInstallTarget) []fileGroup {
	return buildSkillFileGroups(target.skillDir, target.includeAgentsMeta)
}

func resolveInstallBaseDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return homeDir, nil
}

func buildSkillFileGroups(skillDir string, includeAgentsMetadata bool) []fileGroup {
	groups := []fileGroup{
		{
			templateDir: "agents/templates/skill",
			destDir:     skillDir,
			files:       skillRootFiles,
			replaceVer:  true,
		},
		{
			templateDir: "agents/templates/skill/commands",
			destDir:     filepath.Join(skillDir, "commands"),
			files:       skillCommandFiles,
		},
		{
			templateDir: "agents/templates/skill/workflows",
			destDir:     filepath.Join(skillDir, "workflows"),
			files:       skillWorkflowFiles,
		},
		{
			templateDir: "agents/templates/skill/scripts",
			destDir:     filepath.Join(skillDir, "scripts"),
			files:       skillScriptFiles,
			executable:  true,
		},
		{
			templateDir: "agents/templates/subagents",
			destDir:     filepath.Join(skillDir, "subagents"),
			files:       subagentFiles,
		},
	}

	if includeAgentsMetadata {
		groups = append(groups, fileGroup{
			templateDir: "agents/templates/skill/agents",
			destDir:     filepath.Join(skillDir, "agents"),
			files:       []string{"openai.yaml"},
		})
	}

	return groups
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

// skillRenderFunc transforms raw template content into format-specific skill content.
type skillRenderFunc func(content []byte, skillName string) ([]byte, error)

const commandTemplateDir = "agents/templates/commands"

// installCommandSkills installs command templates as individual skills using the given render function.
func installCommandSkills(baseDir, skillsBaseDir string, files []string, render skillRenderFunc) error {
	for _, filename := range files {
		skillName := strings.TrimSuffix(filename, ".md")
		content, err := agentTemplates.ReadFile(commandTemplateDir + "/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", filename, err)
		}

		content, err = render(content, skillName)
		if err != nil {
			return fmt.Errorf("failed to render skill for %s: %w", skillName, err)
		}

		destDir := filepath.Join(baseDir, skillsBaseDir, skillName)
		if err := os.MkdirAll(destDir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}
		if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), content, 0600); err != nil {
			return fmt.Errorf("failed to write SKILL.md for %s: %w", skillName, err)
		}
	}
	return nil
}

// parseFrontmatter splits content into frontmatter lines and body, injecting the skill name.
// Returns the rendered frontmatter lines (without --- markers), the body, and any argument-hint value found.
func parseFrontmatter(content []byte, name string) (frontmatterLines []string, body []byte, argumentHint string, err error) {
	frontmatter, bodyBytes, found := splitFrontmatter(content)
	if !found {
		return nil, nil, "", fmt.Errorf("missing frontmatter")
	}

	lines := strings.Split(string(frontmatter), "\n")
	rendered := make([]string, 0, len(lines)+1)
	rendered = append(rendered, "name: "+name)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "name:") {
			continue
		}
		if strings.HasPrefix(trimmed, "argument-hint:") {
			argumentHint = strings.TrimSpace(strings.TrimPrefix(trimmed, "argument-hint:"))
			continue
		}
		rendered = append(rendered, line)
	}

	return rendered, bodyBytes, argumentHint, nil
}

// assembleFrontmatter joins frontmatter lines, validates YAML, and assembles the full document.
func assembleFrontmatter(frontmatterLines []string, body []byte) ([]byte, error) {
	frontmatterBlock := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatterBlock), &map[string]any{}); err != nil {
		return nil, fmt.Errorf("invalid generated frontmatter: %w", err)
	}

	var result strings.Builder
	result.WriteString("---\n")
	result.WriteString(frontmatterBlock)
	result.WriteString("\n---\n")
	result.Write(body)

	return []byte(result.String()), nil
}

func renderClaudeSkillContent(content []byte, name string) ([]byte, error) {
	frontmatterLines, body, argumentHint, err := parseFrontmatter(content, name)
	if err != nil {
		return nil, err
	}

	// Claude keeps argument-hint in frontmatter with proper YAML encoding.
	if argumentHint != "" {
		frontmatterLines = append(frontmatterLines, "argument-hint: "+encodeYAMLScalar(argumentHint))
	}

	return assembleFrontmatter(frontmatterLines, body)
}

func renderCodexSkillContent(content []byte, name string) ([]byte, error) {
	frontmatterLines, body, argumentHint, err := parseFrontmatter(content, name)
	if err != nil {
		return nil, err
	}

	// Codex replaces $ARGUMENTS in the body instead of keeping argument-hint in frontmatter.
	bodyText := string(body)
	if strings.Contains(bodyText, "$ARGUMENTS") {
		replacement := "If the user included explicit arguments in their request, honor them."
		if argumentHint != "" {
			replacement += fmt.Sprintf(" Expected argument shape: `%s`.", argumentHint)
		}
		bodyText = strings.Replace(bodyText, "$ARGUMENTS", replacement, 1)
	}

	return assembleFrontmatter(frontmatterLines, []byte(bodyText))
}

func encodeYAMLScalar(raw string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range raw {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func cleanupOldCommandFiles(baseDir string, files []string) {
	commandsDir := filepath.Join(baseDir, ".claude", "commands")
	for _, filename := range files {
		_ = os.Remove(filepath.Join(commandsDir, filename))
	}
}

func splitFrontmatter(content []byte) (frontmatter, body []byte, found bool) {
	marker := []byte("---\n")
	if !bytes.HasPrefix(content, marker) {
		return nil, nil, false
	}

	rest := content[len(marker):]
	idx := bytes.Index(rest, marker)
	if idx < 0 {
		return nil, nil, false
	}

	frontmatter = rest[:idx]
	body = rest[idx+len(marker):]
	return frontmatter, body, true
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

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func isAnySkillInstalled(baseDir string) bool {
	for _, skillPath := range installedSkillManifestPaths(baseDir) {
		if _, err := os.Stat(skillPath); err == nil {
			return true
		}
	}
	return false
}

func installedSkillManifestPaths(baseDir string) []string {
	paths := installedSkillManifestPathsForFormat(baseDir, agentSkillFormatClaude)
	paths = append(paths, installedSkillManifestPathsForFormat(baseDir, agentSkillFormatCodex)...)
	return paths
}

func installedSkillManifestPathsForFormat(baseDir string, format agentSkillFormat) []string {
	switch format {
	case agentSkillFormatCodex:
		return []string{
			filepath.Join(baseDir, ".codex", "skills", "stackit", "SKILL.md"),
			filepath.Join(baseDir, ".codex", "skills", "stackit", "skill.md"), // legacy path compatibility
		}
	default:
		return []string{
			filepath.Join(baseDir, ".claude", "skills", "stackit", "SKILL.md"),
			filepath.Join(baseDir, ".claude", "skills", "stackit", "skill.md"), // legacy path compatibility
		}
	}
}

func targetsForFormats(formats []agentSkillFormat) []agentInstallTarget {
	targets := make([]agentInstallTarget, 0, len(formats))
	for _, format := range formats {
		targets = append(targets, installTargetForFormat(format))
	}
	return targets
}
