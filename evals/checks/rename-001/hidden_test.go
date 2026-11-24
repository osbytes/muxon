package rename

import (
	"os"
	"strings"
	"testing"
)

func TestRenamed(t *testing.T) {
	b, err := os.ReadFile("oldname.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "func OldName") {
		t.Fatal("OldName remains")
	}
	if NewName() != "new" {
		t.Fatalf("NewName() = %q", NewName())
	}
}
