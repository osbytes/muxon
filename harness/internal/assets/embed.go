package assets

import "embed"

// ModelsYAML is the embedded default models manifest.
//
//go:embed models.yaml
var ModelsYAML []byte

// PromptFS holds versioned prompt templates.
//
//go:embed prompts/compact/*.tmpl prompts/standard/*.tmpl
var PromptFS embed.FS
