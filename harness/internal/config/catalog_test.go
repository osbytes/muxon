package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogOverridesProviderBaseURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("THEE_HOME", home)
	catPath := filepath.Join(home, "catalog.json")
	cat := `{
  "providers": {
    "deepinfra": {
      "base_url": "https://custom.example/v1/openai",
      "api_key_env": "CUSTOM_KEY"
    }
  }
}`
	if err := os.WriteFile(catPath, []byte(cat), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadMergedManifest()
	if err != nil {
		t.Fatal(err)
	}
	p := m.Providers["deepinfra"]
	if p.BaseURL != "https://custom.example/v1/openai" {
		t.Fatalf("base_url %q", p.BaseURL)
	}
	if p.APIKeyEnv != "CUSTOM_KEY" {
		t.Fatalf("api_key_env %q", p.APIKeyEnv)
	}
}
