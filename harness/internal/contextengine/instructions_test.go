package contextengine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentsPrecedenceOverConventions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("from agents"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CONVENTIONS.md"), []byte("from conv"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadProjectInstructions(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if s != "from agents" {
		t.Fatalf("got %q", s)
	}
}
