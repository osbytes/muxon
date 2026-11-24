package config

import (
	"os"
	"path/filepath"
)

const defaultHome = ".muxon"

// MuxonHome returns MUXON_HOME or ~/.muxon.
func MuxonHome() (string, error) {
	if v := os.Getenv("MUXON_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultHome), nil
}
