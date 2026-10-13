package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollapseBulkDir(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "a.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	count := CountDirEntries(filepath.Join(dir, "node_modules"))
	if count < 1 {
		t.Fatalf("count %d", count)
	}
	collapsed := CollapseBulkEntries([]ListEntry{{Name: "node_modules/", Count: count}})
	if len(collapsed) != 1 || !collapsed[0].Collapsed {
		t.Fatalf("got %+v", collapsed)
	}
}
