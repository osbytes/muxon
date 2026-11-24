package impl

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
	"github.com/PLACEHOLDER/muxon/harness/internal/workspace"
)

func TestGitWorktreeUnavailable(t *testing.T) {
	ws, err := workspace.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"action": "worktree"})
	res, err := NewGit().Run(context.Background(), &tools.Env{Workspace: ws, Headless: true}, args)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !strings.Contains(res.Text, "not available") {
		t.Fatalf("%+v", res)
	}
}

func TestGitCommitHeadlessDenied(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--template=")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]interface{}{"action": "commit", "message": "x", "paths": []string{"a.txt"}})
	res, err := NewGit().Run(context.Background(), &tools.Env{Workspace: ws, Headless: true}, args)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("commit should be disabled")
	}
}
