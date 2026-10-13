package verifier

import (
	"os"
	"path/filepath"
)

// ResolveCommand picks a default verify command for the workspace.
func ResolveCommand(workspaceRoot string) ([]string, bool) {
	if hasGoMod(workspaceRoot) {
		return []string{"go", "test", "./..."}, true
	}
	return nil, false
}

func hasGoMod(root string) bool {
	st, err := os.Stat(filepath.Join(root, "go.mod"))
	return err == nil && !st.IsDir()
}
