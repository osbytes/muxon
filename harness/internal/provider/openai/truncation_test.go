package openai

import "testing"

func TestTruncatedByServer(t *testing.T) {
	if !TruncatedByServer(1000, 400) {
		t.Fatal("expected truncated")
	}
	if TruncatedByServer(100, 90) {
		t.Fatal("not truncated")
	}
}
