package verifier

import "testing"

func TestProtectedTestPathGo(t *testing.T) {
	if !ProtectedTestPath("sum_test.go") {
		t.Fatal("expected protected")
	}
	if ProtectedTestPath("sum.go") {
		t.Fatal("not protected")
	}
}
