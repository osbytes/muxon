package impl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
	"github.com/PLACEHOLDER/thee/harness/internal/workspace"
)

func TestReadSlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := &tools.Env{Workspace: ws}
	args, _ := json.Marshal(map[string]interface{}{"path": "f.txt", "start_line": 2, "end_line": 2})
	res, err := NewRead().Run(context.Background(), env, args)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Text != "b" {
		t.Fatalf("got %+v", res)
	}
}
