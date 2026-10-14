package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type uiState struct {
	LastProvider string `json:"last_provider,omitempty"`
	LastModel    string `json:"last_model,omitempty"`
}

func statePath() (string, error) {
	home, err := TheeHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "state.json"), nil
}

func loadUIState() (uiState, error) {
	path, err := statePath()
	if err != nil {
		return uiState{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return uiState{}, nil
		}
		return uiState{}, err
	}
	var s uiState
	if err := json.Unmarshal(b, &s); err != nil {
		return uiState{}, err
	}
	return s, nil
}

func saveUIState(s uiState) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// LastProvider returns the provider id the interactive session last connected.
func LastProvider() string {
	s, err := loadUIState()
	if err != nil {
		return ""
	}
	return s.LastProvider
}

// LastModel returns the worker model last selected for LastProvider.
func LastModel() string {
	s, err := loadUIState()
	if err != nil {
		return ""
	}
	return s.LastModel
}

// SaveLastProvider records the connected provider id. It never stores a key.
// Switching to a different provider clears the saved model.
func SaveLastProvider(id string) error {
	s, err := loadUIState()
	if err != nil {
		return err
	}
	if s.LastProvider != id {
		s.LastModel = ""
	}
	s.LastProvider = id
	return saveUIState(s)
}

// SaveLastModel records the worker model for the current provider.
func SaveLastModel(model string) error {
	s, err := loadUIState()
	if err != nil {
		return err
	}
	s.LastModel = model
	return saveUIState(s)
}
