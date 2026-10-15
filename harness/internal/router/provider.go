package router

import (
	"fmt"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/openai"
	"github.com/PLACEHOLDER/thee/harness/internal/session"
)

// NewProvider builds a provider client for a binding.
func NewProvider(b *Binding) (provider.Provider, error) {
	switch b.Provider.API {
	case "openai_chat", "":
		key, err := config.ResolveProviderKey(b.Provider.APIKeyEnv, b.ProviderName)
		if err != nil {
			return nil, err
		}
		client := openai.NewClient(b.Provider.BaseURL, key)
		if strings.Contains(b.Provider.BaseURL, "opencode.ai") {
			client.Headers = map[string]string{"x-opencode-session": session.NewID()}
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported provider api %q", b.Provider.API)
	}
}

// NewProviderForRole resolves role and returns provider plus binding.
func NewProviderForRole(m *config.Manifest, role string, workerModelOverride string) (provider.Provider, *Binding, error) {
	var b *Binding
	var err error
	if role == "worker" {
		b, err = ResolveWorker(m, workerModelOverride)
	} else {
		b, err = ResolveRole(m, role)
	}
	if err != nil {
		return nil, nil, err
	}
	p, err := NewProvider(b)
	if err != nil {
		return nil, nil, err
	}
	return p, b, nil
}
