package session

import (
	"regexp"
	"testing"
)

func TestNewIDFormat(t *testing.T) {
	id := NewID()
	if !regexp.MustCompile(`^\d{8}T\d{6}Z-[0-9a-f]{6}$`).MatchString(id) {
		t.Fatalf("NewID() = %q", id)
	}
}
