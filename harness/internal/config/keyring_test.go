package config

import "testing"

func TestKeyringAfterEnv(t *testing.T) {
	mem := NewMemoryKeyring()
	SetSecretStore(mem)
	t.Cleanup(func() { SetSecretStore(nil) })
	if err := StoreProviderKey("deepinfra", "secret-value"); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveProviderKey("MISSING_THEE_KEY", "deepinfra")
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret-value" {
		t.Fatal(got)
	}
	t.Setenv("MISSING_THEE_KEY", "from-env")
	got, err = ResolveProviderKey("MISSING_THEE_KEY", "deepinfra")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-env" {
		t.Fatal(got)
	}
}
