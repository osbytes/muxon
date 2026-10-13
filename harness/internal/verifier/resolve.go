package verifier

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveOptions configures command resolution.
type ResolveOptions struct {
	CLIOverride string
	ConfigCmd   string
}

// ResolveCommand picks a default verify command for the workspace.
func ResolveCommand(workspaceRoot string) ([]string, bool) {
	return Resolve(workspaceRoot, ResolveOptions{})
}

// Resolve picks verify argv using spec precedence.
func Resolve(workspaceRoot string, opt ResolveOptions) ([]string, bool) {
	if opt.CLIOverride != "" {
		return splitCmd(opt.CLIOverride), true
	}
	if opt.ConfigCmd != "" {
		return splitCmd(opt.ConfigCmd), true
	}
	if cmd := npmTest(workspaceRoot); cmd != nil {
		return cmd, true
	}
	if cmd := makeTest(workspaceRoot); cmd != nil {
		return cmd, true
	}
	if hasPytest(workspaceRoot) {
		return []string{"pytest", "-q"}, true
	}
	if hasGoMod(workspaceRoot) {
		return []string{"go", "test", "./..."}, true
	}
	if hasCargo(workspaceRoot) {
		return []string{"cargo", "test"}, true
	}
	return nil, false
}

func splitCmd(s string) []string {
	return strings.Fields(s)
}

func npmTest(root string) []string {
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(root, "pnpm-lock.yaml")); err == nil {
		return []string{"pnpm", "test"}
	}
	if _, err := os.Stat(filepath.Join(root, "yarn.lock")); err == nil {
		return []string{"yarn", "test"}
	}
	return []string{"npm", "test"}
}

func makeTest(root string) []string {
	mk := filepath.Join(root, "Makefile")
	b, err := os.ReadFile(mk)
	if err != nil {
		return nil
	}
	if strings.Contains(string(b), "test:") || strings.Contains(string(b), "test\n") {
		return []string{"make", "test"}
	}
	return nil
}

func hasPytest(root string) bool {
	for _, name := range []string{"pytest.ini", "pyproject.toml", "setup.cfg", "tox.ini"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func hasGoMod(root string) bool {
	st, err := os.Stat(filepath.Join(root, "go.mod"))
	return err == nil && !st.IsDir()
}

func hasCargo(root string) bool {
	_, err := os.Stat(filepath.Join(root, "Cargo.toml"))
	return err == nil
}

// CommandExists checks whether argv[0] is on PATH.
func CommandExists(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	_, err := exec.LookPath(argv[0])
	return err == nil
}
