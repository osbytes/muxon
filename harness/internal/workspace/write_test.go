package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaleReadMessageWithLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ws.NoteReadRange("f.txt", []byte("line\n"), 1, 1)
	if err := os.WriteFile(path, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = ws.CheckFresh("f.txt")
	if err == nil || !contains(err.Error(), "re-read lines 1–1") {
		t.Fatalf("got %v", err)
	}
}

func TestCheckFreshAllowsMissingFile(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.CheckFresh("new.go"); err != nil {
		t.Fatalf("CheckFresh(new.go) = %v, want nil", err)
	}
	if err := ws.WriteFile("new.go", []byte("package new\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "new.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "package new\n" {
		t.Fatalf("new.go = %q, want created contents", got)
	}
}

func contains(s, sub string) bool {
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
