package impl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
	"github.com/PLACEHOLDER/muxon/harness/internal/workspace"
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
	env := &tools.Env{Workspace: ws, Trusted: true}
	args, _ := json.Marshal(map[string]interface{}{"path": "f.txt", "start_line": 2, "end_line": 2})
	res, err := NewRead().Run(context.Background(), env, args)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !strings.Contains(res.Text, "b") {
		t.Fatalf("got %+v", res)
	}
}

func TestReadDeniesEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := &tools.Env{Workspace: ws, Trusted: true, Policy: security.NewPolicy()}
	args, _ := json.Marshal(map[string]interface{}{"path": ".env", "start_line": 1, "end_line": 1})
	res, err := NewRead().Run(context.Background(), env, args)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !strings.Contains(res.Text, security.DenyReadMessage) {
		t.Fatalf("got %+v", res)
	}
}
