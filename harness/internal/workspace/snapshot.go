package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// EnsureSessionBase records the workspace state at session start.
func EnsureSessionBase(root, sessionID string) error {
	git := filepath.Join(root, ".git")
	if _, err := os.Stat(git); err == nil {
		ref := fmt.Sprintf("refs/muxon/sessions/%s/base", sessionID)
		return writeShadowRef(root, ref)
	}
	dest := filepath.Join(root, ".muxon", "sessions", sessionID, "snapshots", "base")
	return os.MkdirAll(dest, 0o755)
}

func writeShadowRef(root, ref string) error {
	idx := filepath.Join(root, ".muxon", "tmp-index")
	_ = os.Remove(idx)
	env := append(os.Environ(), "GIT_INDEX_FILE="+idx)
	add := exec.Command("git", "add", "-A")
	add.Dir = root
	add.Env = env
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %v %s", err, out)
	}
	tree := exec.Command("git", "write-tree")
	tree.Dir = root
	tree.Env = env
	treeOut, err := tree.Output()
	if err != nil {
		return err
	}
	treeHash := string(treeOut[:40])
	commit := exec.Command("git", "commit-tree", treeHash, "-m", "muxon session base")
	commit.Dir = root
	commit.Env = env
	commitOut, err := commit.Output()
	if err != nil {
		return err
	}
	commitHash := string(commitOut[:40])
	update := exec.Command("git", "update-ref", ref, commitHash)
	update.Dir = root
	if out, err := update.CombinedOutput(); err != nil {
		return fmt.Errorf("update-ref: %v %s", err, out)
	}
	return nil
}
