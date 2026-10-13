package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Workspace is the harness working tree.
type Workspace struct {
	root  string
	track *changeTracker
}

// Open validates and opens a workspace directory.
func Open(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("workspace is not a directory: %s", abs)
	}
	return &Workspace{root: abs, track: newChangeTracker()}, nil
}

// Root returns the absolute workspace root.
func (w *Workspace) Root() string {
	return w.root
}

// Resolve maps a relative path inside the workspace.
func (w *Workspace) Resolve(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", rel)
	}
	clean := filepath.Clean(rel)
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", rel)
	}
	abs := filepath.Join(w.root, clean)
	relToRoot, err := filepath.Rel(w.root, abs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(relToRoot, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", rel)
	}
	return abs, nil
}

// Changed reports whether the harness modified any file.
func (w *Workspace) Changed() bool {
	return w.track.changed()
}
