package feature

import "testing"

func TestEnabled(t *testing.T) {
	if !Enabled() {
		t.Fatal("want enabled")
	}
}
