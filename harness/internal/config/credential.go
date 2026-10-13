package config

import (
	"fmt"
	"os"
)

// ResolveAPIKey reads the named environment variable for a provider key.
func ResolveAPIKey(envName string) (string, error) {
	if envName == "" {
		return "", fmt.Errorf("api_key_env not configured")
	}
	v := os.Getenv(envName)
	if v == "" {
		return "", fmt.Errorf("missing API key: set %s", envName)
	}
	return v, nil
}
