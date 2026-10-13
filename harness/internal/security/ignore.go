package security

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

var bulkDirNames = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".git":         true,
	".venv":        true,
}

// IgnoreRules applies gitignore-style rules and bulk-dir skipping.
type IgnoreRules struct {
	git    *ignore.GitIgnore
	thee   *ignore.GitIgnore
	root   string
	policy *Policy
}

// LoadIgnore loads .gitignore and .theeignore from the workspace root.
func LoadIgnore(root string, policy *Policy) (*IgnoreRules, error) {
	r := &IgnoreRules{root: root, policy: policy}
	r.git = compileIgnoreFile(filepath.Join(root, ".gitignore"))
	r.thee = compileIgnoreFile(filepath.Join(root, ".theeignore"))
	return r, nil
}

func compileIgnoreFile(path string) *ignore.GitIgnore {
	b, err := os.ReadFile(path)
	if err != nil {
		return ignore.CompileIgnoreLines()
	}
	lines := strings.Split(string(b), "\n")
	return ignore.CompileIgnoreLines(lines...)
}

// IsBulkDir reports a directory name that should be collapsed or skipped.
func IsBulkDir(name string) bool {
	return bulkDirNames[name]
}

// SkipInSearchList returns true when search/list should not traverse or return the path.
func (r *IgnoreRules) SkipInSearchList(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return false
	}
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if IsBulkDir(p) {
			return true
		}
	}
	if r.policy != nil && r.policy.DenyRead(rel) {
		return true
	}
	if r.matchesIgnore(rel) {
		return true
	}
	return false
}

func (r *IgnoreRules) matchesIgnore(rel string) bool {
	if r.git != nil && r.git.MatchesPath(rel) {
		return true
	}
	if r.thee != nil && r.thee.MatchesPath(rel) {
		return true
	}
	return false
}

// ListEntry is one row for the list tool.
type ListEntry struct {
	Name      string
	Collapsed bool
	Count     int
}

// CollapseBulkEntries replaces bulk directories with a single collapsed row.
func CollapseBulkEntries(entries []ListEntry) []ListEntry {
	out := make([]ListEntry, 0, len(entries))
	for _, e := range entries {
		if e.Collapsed {
			out = append(out, e)
			continue
		}
		base := filepath.Base(strings.TrimSuffix(e.Name, "/"))
		if IsBulkDir(base) {
			out = append(out, ListEntry{Name: e.Name + " (" + itoa(e.Count) + " items)", Collapsed: true, Count: e.Count})
			continue
		}
		out = append(out, e)
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// CountDirEntries counts files under a directory (for collapse label).
func CountDirEntries(absDir string) int {
	n := 0
	_ = filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == absDir {
			return nil
		}
		n++
		if d.IsDir() && IsBulkDir(d.Name()) {
			return filepath.SkipDir
		}
		return nil
	})
	return n
}
