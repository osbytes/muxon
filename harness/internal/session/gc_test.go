package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGCRemovesOldSessions(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, ".muxon", "sessions", "old")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(root, ".muxon", "sessions", "fresh")
	if err := os.MkdirAll(fresh, 0o755); err != nil {
		t.Fatal(err)
	}
	n, err := GC(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("removed %d", n)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old session remains")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal(err)
	}
}
