package security

import (
	"path/filepath"
	"testing"
)

func TestTrustAddList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MUXON_HOME", home)
	dir := filepath.Join(home, "proj")
	s := TrustStore{}
	s.AddRoot(dir)
	if err := SaveTrust(s); err != nil {
		t.Fatal(err)
	}
	if !IsTrustedRoot(dir) {
		t.Fatal("expected trusted")
	}
}
