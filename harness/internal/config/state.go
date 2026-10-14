package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type uiState struct {
	LastProvider string `json:"last_provider,omitempty"`
}

func statePath() (string, error) {
	home, err := TheeHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "state.json"), nil
}

// LastProvider returns the provider id the interactive session last connected.
func LastProvider() string {
	path, err := statePath()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var s uiState
	if err := json.Unmarshal(b, &s); err != nil {
		return ""
	}
	return s.LastProvider
}

// SaveLastProvider records the connected provider id. It never stores a key.
func SaveLastProvider(id string) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(uiState{LastProvider: id}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
