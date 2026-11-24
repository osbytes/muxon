package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"
)

func TestTrustAddListRm(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MUXON_HOME", home)
	root := filepath.Join(home, "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := security.LoadTrust()
	if err != nil {
		t.Fatal(err)
	}
	s.AddRootWithConfigHash(root, "abc")
	if err := security.SaveTrust(s); err != nil {
		t.Fatal(err)
	}
	s2, err := security.LoadTrust()
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.Roots) != 1 {
		t.Fatalf("%v", s2.Roots)
	}
	s2.RemoveRoot(root)
	if err := security.SaveTrust(s2); err != nil {
		t.Fatal(err)
	}
	s3, _ := security.LoadTrust()
	if len(s3.Roots) != 0 {
		t.Fatalf("%v", s3.Roots)
	}
	_ = bytes.NewBuffer(nil)
}
