package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type readMeta struct {
	content   []byte
	startLine int
	endLine   int
}

type changeTracker struct {
	original map[string][]byte
	current  map[string][]byte
	readMeta map[string]readMeta
	revision int
}

func newChangeTracker() *changeTracker {
	return &changeTracker{
		original: make(map[string][]byte),
		current:  make(map[string][]byte),
		readMeta: make(map[string]readMeta),
	}
}

func (c *changeTracker) changed() bool {
	for k, cur := range c.current {
		orig, ok := c.original[k]
		if !ok || !bytesEqual(orig, cur) {
			return true
		}
	}
	return false
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ReadHash returns a content hash for a resolved absolute path.
func (w *Workspace) ReadHash(abs string) (string, error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// NoteRead records content at read time for stale detection.
func (w *Workspace) NoteRead(rel string, content []byte) {
	w.NoteReadRange(rel, content, 0, 0)
}

// NoteReadRange records content and the line range the model last read.
func (w *Workspace) NoteReadRange(rel string, content []byte, startLine, endLine int) {
	w.track.original[rel] = append([]byte(nil), content...)
	w.track.current[rel] = append([]byte(nil), content...)
	w.track.readMeta[rel] = readMeta{content: append([]byte(nil), content...), startLine: startLine, endLine: endLine}
}

// WriteFile atomically writes data to a workspace-relative path.
func (w *Workspace) WriteFile(rel string, data []byte) error {
	abs, err := w.Resolve(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	tmp, err := os.CreateTemp(dir, ".thee-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if _, ok := w.track.original[rel]; !ok {
		if orig, err := os.ReadFile(abs); err == nil {
			w.track.original[rel] = append([]byte(nil), orig...)
		}
	}
	if err := os.Rename(tmpName, abs); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	w.track.current[rel] = append([]byte(nil), data...)
	w.NoteMutation()
	return nil
}

// Revision is how many times this workspace has been written or deleted.
func (w *Workspace) Revision() int {
	return w.track.revision
}

// NoteMutation records a successful write or delete.
func (w *Workspace) NoteMutation() {
	w.track.revision++
}

// ReadFile reads a workspace-relative file.
func (w *Workspace) ReadFile(rel string) ([]byte, error) {
	abs, err := w.Resolve(rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

// CheckFresh returns an error if the file changed since NoteRead.
func (w *Workspace) CheckFresh(rel string) error {
	abs, err := w.Resolve(rel)
	if err != nil {
		return err
	}
	cur, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	seen, ok := w.track.original[rel]
	if !ok {
		return nil
	}
	if !bytesEqual(seen, cur) {
		meta := w.track.readMeta[rel]
		if meta.startLine > 0 && meta.endLine >= meta.startLine {
			return fmt.Errorf("file changed since you read it; re-read lines %d–%d", meta.startLine, meta.endLine)
		}
		return fmt.Errorf("file changed since you read it; re-read %s", rel)
	}
	return nil
}
