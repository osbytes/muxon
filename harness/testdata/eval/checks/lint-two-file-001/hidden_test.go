package lint

import "testing"

func TestConsistent(t *testing.T) {
	if A() != B() {
		t.Fatalf("A=%d B=%d", A(), B())
	}
	if A() != 1 {
		t.Fatalf("A=%d", A())
	}
}
