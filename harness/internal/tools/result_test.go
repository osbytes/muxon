package tools

import "testing"

func TestFormatResultRedacts(t *testing.T) {
	body := "DEEPINFRA_API_KEY=sk-abcdefghijklmnopqrstuvwxyz"
	out := FormatResult("read", true, body, 0, 0)
	if contains(out, "sk-abcdefghijklmnopqrst") {
		t.Fatalf("secret leaked: %s", out)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
