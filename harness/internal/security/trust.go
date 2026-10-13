package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
)

// TrustStore lists trusted workspace roots.
type TrustStore struct {
	Roots []string `json:"roots"`
}

// LoadTrust reads ~/.thee/trust.json.
func LoadTrust() (TrustStore, error) {
	home, err := config.TheeHome()
	if err != nil {
		return TrustStore{}, err
	}
	path := filepath.Join(home, "trust.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TrustStore{}, nil
		}
		return TrustStore{}, err
	}
	var s TrustStore
	if err := json.Unmarshal(b, &s); err != nil {
		return TrustStore{}, err
	}
	return s, nil
}

// SaveTrust writes the trust store.
func SaveTrust(s TrustStore) error {
	home, err := config.TheeHome()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(home, "trust.json"), b, 0o644)
}

// AddRoot adds a workspace root if missing.
func (s *TrustStore) AddRoot(root string) {
	root = filepath.Clean(root)
	for _, r := range s.Roots {
		if filepath.Clean(r) == root {
			return
		}
	}
	s.Roots = append(s.Roots, root)
}

// RemoveRoot removes a workspace root.
func (s *TrustStore) RemoveRoot(root string) {
	root = filepath.Clean(root)
	var out []string
	for _, r := range s.Roots {
		if filepath.Clean(r) != root {
			out = append(out, r)
		}
	}
	s.Roots = out
}

// IsTrustedRoot reports whether the workspace is trusted for verify.
func IsTrustedRoot(workspaceRoot string) bool {
	if os.Getenv("THEE_TRUST") == "1" {
		return true
	}
	s, err := LoadTrust()
	if err != nil {
		return false
	}
	ws := filepath.Clean(workspaceRoot)
	for _, r := range s.Roots {
		r = filepath.Clean(r)
		if ws == r || strings.HasPrefix(ws, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// WorkspaceTrusted combines CLI flag and trust store.
func WorkspaceTrusted(workspaceRoot string, flagTrust bool) bool {
	return flagTrust || IsTrustedRoot(workspaceRoot)
}
