package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectCannotExceedMaxPermission(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	t.Setenv("THEE_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[security]
max_permission = "edit"
permission = "edit"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "thee.toml"), []byte(`
[security]
permission = "full"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadEffective(ws)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Permission != "edit" {
		t.Fatalf("got %q", cfg.Permission)
	}
}
