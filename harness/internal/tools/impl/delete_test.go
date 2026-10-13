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

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rm.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, _ := workspace.Open(dir)
	env := &tools.Env{Workspace: ws, Trusted: true}
	args, _ := json.Marshal(map[string]string{"path": "rm.txt"})
	res, err := NewDelete().Run(context.Background(), env, args)
	if err != nil || !res.OK {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should be gone")
	}
}

func TestDeleteRejectsParent(t *testing.T) {
	dir := t.TempDir()
	ws, _ := workspace.Open(dir)
	env := &tools.Env{Workspace: ws, Trusted: true}
	args, _ := json.Marshal(map[string]string{"path": "../outside"})
	res, _ := NewDelete().Run(context.Background(), env, args)
	if res.OK {
		t.Fatal("expected fail")
	}
}
