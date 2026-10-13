package director

import (
	"io"
	"os"
	"path/filepath"
)

// CheckpointStore keeps file snapshots for non-git restore.
type CheckpointStore struct {
	root      string
	sessionID string
	seq       int
}

// NewCheckpointStore creates a store under .thee/sessions.
func NewCheckpointStore(root, sessionID string) *CheckpointStore {
	return &CheckpointStore{root: root, sessionID: sessionID}
}

// Save copies a file snapshot after a successful patch.
func (c *CheckpointStore) Save(relPath string) error {
	src, err := filepath.Abs(filepath.Join(c.root, relPath))
	if err != nil {
		return err
	}
	c.seq++
	destDir := filepath.Join(c.root, ".thee", "sessions", c.sessionID, "snapshots", "cp", itoa(c.seq))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(destDir, filepath.Base(relPath))
	return copyFile(src, dest)
}

// RestoreLatest puts the latest snapshot back for relPath basename.
func (c *CheckpointStore) RestoreLatest(relPath string) error {
	if c.seq == 0 {
		return os.ErrNotExist
	}
	dest := filepath.Join(c.root, relPath)
	src := filepath.Join(c.root, ".thee", "sessions", c.sessionID, "snapshots", "cp", itoa(c.seq), filepath.Base(relPath))
	return copyFile(src, dest)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [12]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}
