package config

import (
	"os"
	"path/filepath"
)

const defaultHome = ".thee"

// TheeHome returns THEE_HOME or ~/.thee.
func TheeHome() (string, error) {
	if v := os.Getenv("THEE_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultHome), nil
}
