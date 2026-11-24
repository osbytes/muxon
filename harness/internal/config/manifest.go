package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/PLACEHOLDER/muxon/harness/internal/assets"
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
	API       string `yaml:"api" json:"api"`
	BaseURL   string `yaml:"base_url" json:"base_url"`
	APIKeyEnv string `yaml:"api_key_env" json:"api_key_env"`
	MaxConc   int    `yaml:"max_concurrency" json:"max_concurrency"`
}

// RoleDef binds a logical role to a manifest model id.
type RoleDef struct {
	ModelID string `yaml:"model_id"`
}

// ModelDef describes one model entry.
type ModelDef struct {
	ID              string       `yaml:"id" json:"id"`
	Provider        string       `yaml:"provider" json:"provider"`
	Model           string       `yaml:"model" json:"model"`
	InputPerM       float64      `yaml:"input_per_m" json:"input_per_m"`
	OutputPerM      float64      `yaml:"output_per_m" json:"output_per_m"`
	CachedInputPerM float64      `yaml:"cached_input_per_m" json:"cached_input_per_m"`
	CacheWritePerM  float64      `yaml:"cache_write_per_m" json:"cache_write_per_m"`
	ReasoningPerM   float64      `yaml:"reasoning_per_m" json:"reasoning_per_m"`
	Capabilities    Capabilities `yaml:"capabilities" json:"capabilities"`
}

// Capabilities holds model capability profile fields.
type Capabilities struct {
	ContextWindow     int    `yaml:"context_window" json:"context_window"`
	ToolProtocol      string `yaml:"tool_protocol" json:"tool_protocol"`
	EditFormat        string `yaml:"edit_format" json:"edit_format"`
	PromptVariant     string `yaml:"prompt_variant" json:"prompt_variant"`
	StructuredOut     string `yaml:"structured_output" json:"structured_output"`
	MaxOutput         int    `yaml:"max_output_tokens" json:"max_output_tokens"`
	ParallelToolCalls bool   `yaml:"parallel_tool_calls" json:"parallel_tool_calls"`
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

// ResolveModel returns a model entry by id.
func (m *Manifest) ResolveModel(id string) (*ModelDef, error) {
	for i := range m.Models {
		if m.Models[i].ID == id {
			return &m.Models[i], nil
		}
	}
	return nil, fmt.Errorf("unknown model id %q", id)
}

// ProviderForModel returns the merged provider definition for a model.
func (m *Manifest) ProviderForModel(md *ModelDef) (ProviderDef, error) {
	p, ok := m.Providers[md.Provider]
	if !ok {
		return ProviderDef{}, fmt.Errorf("unknown provider %q", md.Provider)
	}
	return p, nil
}
