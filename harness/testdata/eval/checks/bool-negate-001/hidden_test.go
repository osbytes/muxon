package bit

import "testing"

func TestNot(t *testing.T) {
	if Not(true) {
		t.Fatal("Not(true)")
	}
	if !Not(false) {
		t.Fatal("Not(false)")
	}
}
