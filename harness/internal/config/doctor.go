package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DoctorReport is a non-secret health summary.
type DoctorReport struct {
	Lines []string
}

// RunDoctor checks manifest roles and local tooling.
func RunDoctor(m *Manifest) DoctorReport {
	var lines []string
	if m == nil {
		lines = append(lines, "manifest: not loaded")
		return DoctorReport{Lines: lines}
	}
	for _, role := range []string{"worker", "router", "escalated"} {
		md, err := m.ResolveRole(role)
		if err != nil {
			continue
		}
		pd, err := m.ProviderForModel(md)
		if err != nil {
			lines = append(lines, fmt.Sprintf("role %s: %v", role, err))
			continue
		}
		keyOK := false
		if pd.APIKeyEnv != "" {
			keyOK = envSet(pd.APIKeyEnv)
			lines = append(lines, fmt.Sprintf("role %s: provider=%s model=%s key_env=%s present=%t", role, md.Provider, md.Model, pd.APIKeyEnv, keyOK))
		} else {
			lines = append(lines, fmt.Sprintf("role %s: provider=%s model=%s (no key required)", role, md.Provider, md.Model))
		}
		if pd.BaseURL != "" {
			lines = append(lines, fmt.Sprintf("  base_url: %s", pd.BaseURL))
		}
	}
	if _, err := exec.LookPath("rg"); err != nil {
		lines = append(lines, "rg: not found on PATH")
	} else {
		lines = append(lines, "rg: ok")
	}
	return DoctorReport{Lines: lines}
}

func envSet(name string) bool {
	return strings.TrimSpace(os.Getenv(name)) != ""
}
