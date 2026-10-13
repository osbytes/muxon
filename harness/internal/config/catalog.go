package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Catalog is the user-level provider registry (catalog.json).
type Catalog struct {
	Providers map[string]ProviderDef `json:"providers"`
	Models    []ModelDef             `json:"models"`
}

// CatalogPath returns the catalog file path from user config or THEE_HOME default.
func CatalogPath() (string, error) {
	home, err := TheeHome()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, "catalog.json")
	uc, err := LoadUserConfig()
	if err == nil && uc.Providers.Catalog != "" {
		path = expandHome(uc.Providers.Catalog, home)
	}
	return path, nil
}

func expandHome(p, home string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		return filepath.Join(home, p[2:])
	}
	return p
}

// LoadCatalog reads catalog.json when present.
func LoadCatalog() (Catalog, error) {
	path, err := CatalogPath()
	if err != nil {
		return Catalog{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Catalog{}, nil
		}
		return Catalog{}, err
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// LoadMergedManifest embeds models.yaml and merges catalog overrides.
func LoadMergedManifest() (*Manifest, error) {
	base, err := LoadEmbeddedManifest()
	if err != nil {
		return nil, err
	}
	cat, err := LoadCatalog()
	if err != nil {
		return nil, err
	}
	return MergeCatalog(base, cat), nil
}

// MergeCatalog overlays catalog providers and models onto base.
func MergeCatalog(base *Manifest, cat Catalog) *Manifest {
	out := *base
	out.Providers = map[string]ProviderDef{}
	for k, v := range base.Providers {
		out.Providers[k] = v
	}
	if out.Providers == nil {
		out.Providers = map[string]ProviderDef{}
	}
	for id, p := range cat.Providers {
		existing := out.Providers[id]
		if p.API != "" {
			existing.API = p.API
		}
		if p.BaseURL != "" {
			existing.BaseURL = p.BaseURL
		}
		if p.APIKeyEnv != "" {
			existing.APIKeyEnv = p.APIKeyEnv
		}
		if p.MaxConc != 0 {
			existing.MaxConc = p.MaxConc
		}
		out.Providers[id] = existing
	}
	for _, cm := range cat.Models {
		found := false
		for i := range out.Models {
			if out.Models[i].ID == cm.ID {
				out.Models[i] = mergeModel(out.Models[i], cm)
				found = true
				break
			}
		}
		if !found {
			ApplyDefaults(&cm.Capabilities)
			out.Models = append(out.Models, cm)
		}
	}
	return &out
}

func mergeModel(base, over ModelDef) ModelDef {
	if over.Provider != "" {
		base.Provider = over.Provider
	}
	if over.Model != "" {
		base.Model = over.Model
	}
	if over.InputPerM != 0 {
		base.InputPerM = over.InputPerM
	}
	if over.OutputPerM != 0 {
		base.OutputPerM = over.OutputPerM
	}
	if over.CachedInputPerM != 0 {
		base.CachedInputPerM = over.CachedInputPerM
	}
	if over.CacheWritePerM != 0 {
		base.CacheWritePerM = over.CacheWritePerM
	}
	if over.ReasoningPerM != 0 {
		base.ReasoningPerM = over.ReasoningPerM
	}
	base.Capabilities = mergeCaps(base.Capabilities, over.Capabilities)
	ApplyDefaults(&base.Capabilities)
	return base
}

func mergeCaps(base, over Capabilities) Capabilities {
	if over.ContextWindow != 0 {
		base.ContextWindow = over.ContextWindow
	}
	if over.ToolProtocol != "" {
		base.ToolProtocol = over.ToolProtocol
	}
	if over.EditFormat != "" {
		base.EditFormat = over.EditFormat
	}
	if over.PromptVariant != "" {
		base.PromptVariant = over.PromptVariant
	}
	if over.StructuredOut != "" {
		base.StructuredOut = over.StructuredOut
	}
	if over.MaxOutput != 0 {
		base.MaxOutput = over.MaxOutput
	}
	return base
}
