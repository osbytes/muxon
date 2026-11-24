package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/zalando/go-keyring"
)

const keyringService = "muxon"

// SecretStore saves provider keys outside the repo and traces.
type SecretStore interface {
	Set(service, user, secret string) error
	Get(service, user string) (string, error)
}

type osKeyring struct{}

func (osKeyring) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

func (osKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

type memoryKeyring struct {
	mu sync.Mutex
	m  map[string]string
}

func (m *memoryKeyring) Set(service, user, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m == nil {
		m.m = map[string]string{}
	}
	m.m[service+"\x00"+user] = secret
	return nil
}

func (m *memoryKeyring) Get(service, user string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.m[service+"\x00"+user]
	if !ok || v == "" {
		return "", fmt.Errorf("secret not found")
	}
	return v, nil
}

var secrets SecretStore = osKeyring{}

// SetSecretStore replaces the keyring. Tests must not call the OS keychain.
func SetSecretStore(s SecretStore) {
	if s == nil {
		secrets = osKeyring{}
		return
	}
	secrets = s
}

// NewMemoryKeyring returns an in-memory store for tests.
func NewMemoryKeyring() SecretStore {
	return &memoryKeyring{}
}

// StoreProviderKey saves a provider key in the keyring.
func StoreProviderKey(providerID, secret string) error {
	if providerID == "" || secret == "" {
		return fmt.Errorf("provider and key are required")
	}
	return secrets.Set(keyringService, providerID, secret)
}

// ResolveProviderKey reads the env var, then the keyring for providerID.
func ResolveProviderKey(envName, providerID string) (string, error) {
	if envName != "" {
		if v := os.Getenv(envName); v != "" {
			return v, nil
		}
	}
	if providerID != "" {
		if v, err := secrets.Get(keyringService, providerID); err == nil && v != "" {
			return v, nil
		}
		v, err := OpenCodeAuthKey(providerID)
		if err != nil {
			return "", err
		}
		if v != "" {
			return v, nil
		}
	}
	if envName == "" {
		return "", fmt.Errorf("missing API key")
	}
	return "", fmt.Errorf("missing API key: set %s", envName)
}
