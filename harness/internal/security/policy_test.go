package security

import "testing"

func TestDenyReadEnv(t *testing.T) {
	p := NewPolicy()
	if !p.DenyRead(".env") {
		t.Fatal(".env should be denied")
	}
	if p.DenyRead(".env.example") {
		t.Fatal(".env.example allowed")
	}
	if !p.DenyRead("secrets/api.key") {
		t.Fatal("key suffix denied")
	}
}
