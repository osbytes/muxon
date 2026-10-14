package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type openCodeCredential struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

// OpenCodeAuthPath is the OpenCode credential file.
// THEE_OPENCODE_AUTH overrides the default path.
func OpenCodeAuthPath() string {
	if p := os.Getenv("THEE_OPENCODE_AUTH"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode", "auth.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "auth.json")
}

// OpenCodeAuthKey reads one provider key from the OpenCode auth file.
// A missing file or provider returns an empty key.
func OpenCodeAuthKey(providerID string) (string, error) {
	path := OpenCodeAuthPath()
	if path == "" || providerID == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var creds map[string]openCodeCredential
	if err := json.Unmarshal(b, &creds); err != nil {
		return "", err
	}
	cred := creds[providerID]
	if cred.Key == "" {
		return "", nil
	}
	return cred.Key, nil
}
