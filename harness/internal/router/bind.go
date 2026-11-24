package router

import (
	"fmt"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
)

// Binding is a resolved role with model and provider metadata.
type Binding struct {
	Role         string
	Model        *config.ModelDef
	Provider     config.ProviderDef
	ProviderName string
}

// ResolveRole binds a manifest role to model and provider.
func ResolveRole(m *config.Manifest, role string) (*Binding, error) {
	md, err := m.ResolveRole(role)
	if err != nil {
		return nil, err
	}
	pd, err := m.ProviderForModel(md)
	if err != nil {
		return nil, err
	}
	if pd.API == "" {
		pd.API = "openai_chat"
	}
	return &Binding{
		Role:         role,
		Model:        md,
		Provider:     pd,
		ProviderName: md.Provider,
	}, nil
}

// ResolveWorker returns the worker binding, optionally overriding model id.
func ResolveWorker(m *config.Manifest, modelIDOverride string) (*Binding, error) {
	if modelIDOverride != "" {
		md, err := m.ResolveModel(modelIDOverride)
		if err != nil {
			return nil, err
		}
		pd, err := m.ProviderForModel(md)
		if err != nil {
			return nil, err
		}
		if pd.API == "" {
			pd.API = "openai_chat"
		}
		return &Binding{
			Role:         "worker",
			Model:        md,
			Provider:     pd,
			ProviderName: md.Provider,
		}, nil
	}
	return ResolveRole(m, "worker")
}

// ResolveEscalated returns the escalated role binding when configured.
func ResolveEscalated(m *config.Manifest) (*Binding, error) {
	if m.Roles == nil || m.Roles["escalated"].ModelID == "" {
		return nil, fmt.Errorf("no escalated role")
	}
	return ResolveRole(m, "escalated")
}
