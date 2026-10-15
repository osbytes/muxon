package feature

import "testing"

func TestEnabledHidden(t *testing.T) {
	if !Enabled() {
		t.Fatal("want enabled")
	}
}
