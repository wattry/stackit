package integrations

import "embed"

// TemplateVersion is the current version of agent templates - set via SetVersion()
var TemplateVersion = "dev"

//go:embed agents/templates/skill/*.md agents/templates/skill/commands/*.md agents/templates/skill/workflows/*.md agents/templates/skill/scripts/*.sh agents/templates/commands/*.md agents/templates/cursor/*.md
var agentTemplates embed.FS
