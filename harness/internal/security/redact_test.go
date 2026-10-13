package security

import "testing"

func TestRedactAPIKey(t *testing.T) {
	in := "DEEPINFRA_API_KEY=sk-abcdefghijklmnop"
	out := RedactSecrets(in)
	if out == in || stringsContains(out, "sk-abcdefghij") {
		t.Fatalf("got %q", out)
	}
}

func stringsContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
