package app

import "testing"

func TestVersion(t *testing.T) {
	if Version != "1" {
		t.Fatalf("Version = %q", Version)
	}
}
