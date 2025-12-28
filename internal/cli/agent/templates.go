package agent

import "embed"

// TemplateVersion is the current version of agent templates - set via SetVersion()
var TemplateVersion = "dev"

//go:embed templates/skill/*.md templates/skill/commands/*.md templates/skill/workflows/*.md templates/skill/scripts/*.sh
var skillTemplates embed.FS

//go:embed templates/commands/*.md
var commandTemplates embed.FS

//go:embed templates/cursor/*.md
var cursorTemplates embed.FS
