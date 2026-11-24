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

// RestoreNewest copies the newest on-disk snapshot for relPath back into the workspace.
func RestoreNewest(root, sessionID, relPath string) error {
	base := filepath.Base(relPath)
	cpRoot := filepath.Join(root, ".thee", "sessions", sessionID, "snapshots", "cp")
	entries, err := os.ReadDir(cpRoot)
	if err != nil {
		return err
	}
	best := ""
	bestN := -1
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := 0
		for _, c := range e.Name() {
			if c < '0' || c > '9' {
				n = -1
				break
			}
			n = n*10 + int(c-'0')
		}
		if n < 0 {
			continue
		}
		cand := filepath.Join(cpRoot, e.Name(), base)
		if _, err := os.Stat(cand); err != nil {
			continue
		}
		if n >= bestN {
			bestN = n
			best = cand
		}
	}
	if best == "" {
		return os.ErrNotExist
	}
	return copyFile(best, filepath.Join(root, relPath))
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
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
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
