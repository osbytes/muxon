package text

import "testing"

func TestClean(t *testing.T) {
	if Clean(" a ") != "a" {
		t.Fatalf("%q", Clean(" a "))
	}
}
