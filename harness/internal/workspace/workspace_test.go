package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRejectsParent(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Resolve("../outside"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAtomicWriteVisible(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("a.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("got %q", b)
	}
	if !ws.Changed() {
		t.Fatal("expected changed")
	}
}
