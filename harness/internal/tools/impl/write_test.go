package impl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
	"github.com/PLACEHOLDER/thee/harness/internal/workspace"
)

func TestWriteCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{
		"path":    "internal/tools/impl/web_search.go",
		"content": "package impl\n",
	})
	res, err := NewWrite().Run(context.Background(), &tools.Env{Workspace: ws, Trusted: true}, args)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !strings.Contains(res.Text, "wrote internal/tools/impl/web_search.go") {
		t.Fatalf("Run() = %+v, want wrote", res)
	}
	got, err := os.ReadFile(filepath.Join(dir, "internal/tools/impl/web_search.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "package impl\n" {
		t.Fatalf("web_search.go = %q", got)
	}
	if !ws.Changed() {
		t.Fatal("Changed() = false, want a new file to count")
	}
}
