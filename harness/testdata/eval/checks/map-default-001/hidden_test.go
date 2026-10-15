package lookup

import "testing"

func TestLabel(t *testing.T) {
	if Label("a") != "alpha" {
		t.Fatalf("%q", Label("a"))
	}
	if Label("z") != "unknown" {
		t.Fatalf("%q", Label("z"))
	}
}
