package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportRedactsKey(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "trace.jsonl")
	if err := os.WriteFile(src, []byte("{\"type\":\"model.response\",\"text\":\"sk-supersecretkey1\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out.jsonl")
	if err := ExportRedacted(src, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "sk-supersecretkey1") {
		t.Fatalf("%s", b)
	}
	if !strings.Contains(string(b), "sk-[REDACTED]") {
		t.Fatalf("%s", b)
	}
}
