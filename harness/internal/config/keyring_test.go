package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyringAfterEnv(t *testing.T) {
	mem := NewMemoryKeyring()
	SetSecretStore(mem)
	t.Cleanup(func() { SetSecretStore(nil) })
	if err := StoreProviderKey("deepinfra", "secret-value"); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveProviderKey("MISSING_MUXON_KEY", "deepinfra")
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret-value" {
		t.Fatal(got)
	}
	t.Setenv("MISSING_MUXON_KEY", "from-env")
	got, err = ResolveProviderKey("MISSING_MUXON_KEY", "deepinfra")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-env" {
		t.Fatal(got)
	}
}

func TestResolveProviderKeyOpenCodeAuth(t *testing.T) {
	SetSecretStore(NewMemoryKeyring())
	t.Cleanup(func() { SetSecretStore(nil) })
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	body := []byte(`{"opencode-go":{"type":"api","key":"from-auth"}}`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MUXON_OPENCODE_AUTH", path)
	t.Setenv("OPENCODE_API_KEY", "")
	got, err := ResolveProviderKey("OPENCODE_API_KEY", "opencode-go")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-auth" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("OPENCODE_API_KEY", "from-env")
	got, err = ResolveProviderKey("OPENCODE_API_KEY", "opencode-go")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-env" {
		t.Fatalf("env should win, got %q", got)
	}
}
