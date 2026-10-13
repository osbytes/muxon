package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/PLACEHOLDER/thee/harness/internal/assets"
)

// Manifest is the parsed models.yaml document.
type Manifest struct {
	Version   int                    `yaml:"version"`
	Providers map[string]ProviderDef `yaml:"providers"`
	Roles     map[string]RoleDef     `yaml:"roles"`
	Models    []ModelDef             `yaml:"models"`
}

// ProviderDef names a provider endpoint and credential env var.
type ProviderDef struct {
	API       string `yaml:"api"`
	BaseURL   string `yaml:"base_url"`
	APIKeyEnv string `yaml:"api_key_env"`
	MaxConc   int    `yaml:"max_concurrency"`
}

// RoleDef binds a logical role to a manifest model id.
type RoleDef struct {
	ModelID string `yaml:"model_id"`
}

// ModelDef describes one model entry.
type ModelDef struct {
	ID           string       `yaml:"id"`
	Provider     string       `yaml:"provider"`
	Model        string       `yaml:"model"`
	Capabilities Capabilities `yaml:"capabilities"`
}

// Capabilities holds model capability profile fields.
type Capabilities struct {
	ContextWindow int    `yaml:"context_window"`
	ToolProtocol  string `yaml:"tool_protocol"`
	EditFormat    string `yaml:"edit_format"`
	PromptVariant string `yaml:"prompt_variant"`
	StructuredOut string `yaml:"structured_output"`
	MaxOutput     int    `yaml:"max_output_tokens"`
}

// LoadEmbeddedManifest parses the shipped models.yaml.
func LoadEmbeddedManifest() (*Manifest, error) {
	return ParseManifest(assets.ModelsYAML)
}

// ParseManifest parses YAML bytes into a Manifest with defaults applied.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for i := range m.Models {
		ApplyDefaults(&m.Models[i].Capabilities)
	}
	return &m, nil
}

// ApplyDefaults fills conservative capability defaults (D6-related defaults).
func ApplyDefaults(c *Capabilities) {
	if c.ContextWindow == 0 {
		c.ContextWindow = 32768
	}
	if c.ToolProtocol == "" {
		c.ToolProtocol = "two_step"
	}
	if c.EditFormat == "" {
		c.EditFormat = "search_replace"
	}
	if c.PromptVariant == "" {
		c.PromptVariant = "compact"
	}
}

// ResolveRole returns the model for a role name.
func (m *Manifest) ResolveRole(role string) (*ModelDef, error) {
	rd, ok := m.Roles[role]
	if !ok {
		return nil, fmt.Errorf("unknown role %q", role)
	}
	for i := range m.Models {
		if m.Models[i].ID == rd.ModelID {
			return &m.Models[i], nil
		}
	}
	return nil, fmt.Errorf("role %q references unknown model_id %q", role, rd.ModelID)
}
