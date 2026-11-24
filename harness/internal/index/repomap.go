package index

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"
)

// Symbol is one indexed name in a file.
type Symbol struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Line int    `json:"line"`
}

// RepoMap is a ranked symbol list on disk.
type RepoMap struct {
	GeneratedAt time.Time `json:"generated_at"`
	Symbols     []Symbol  `json:"symbols"`
}

// CachePath returns the workspace cache file path.
func CachePath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".muxon", "index", "repomap.json")
}

// LoadOrBuild returns ranked symbols, using JSON cache when fresh enough.
func LoadOrBuild(workspaceRoot, goal string, ign *security.IgnoreRules, pol *security.Policy) ([]Symbol, error) {
	cache := CachePath(workspaceRoot)
	if syms, ok := loadCache(cache, workspaceRoot); ok {
		rankSymbols(syms, goal)
		return syms, nil
	}
	syms, err := BuildGoRepoMap(workspaceRoot, goal, ign, pol)
	if err != nil {
		return nil, err
	}
	_ = saveCache(cache, syms)
	return syms, nil
}

func loadCache(cache, root string) ([]Symbol, bool) {
	b, err := os.ReadFile(cache)
	if err != nil {
		return nil, false
	}
	var m RepoMap
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	st, err := os.Stat(cache)
	if err != nil {
		return nil, false
	}
	// Rebuild if any .go file is newer than cache.
	newer := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		fi, err := os.Stat(path)
		if err == nil && fi.ModTime().After(st.ModTime()) {
			newer = true
			return filepath.SkipAll
		}
		return nil
	})
	if newer {
		return nil, false
	}
	return m.Symbols, true
}

func saveCache(cache string, syms []Symbol) error {
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		return err
	}
	m := RepoMap{GeneratedAt: time.Now().UTC(), Symbols: syms}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(cache, b, 0o644)
}

// BuildGoRepoMap walks Go sources and ranks symbols for the goal.
func BuildGoRepoMap(workspaceRoot, goal string, ign *security.IgnoreRules, pol *security.Policy) ([]Symbol, error) {
	var syms []Symbol
	err := filepath.WalkDir(workspaceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if security.IsBulkDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(workspaceRoot, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if pol != nil && pol.DenyRead(rel) {
			return nil
		}
		if ign != nil && ign.SkipInSearchList(rel) {
			return nil
		}
		fileSyms, err := parseGoSymbols(path, rel)
		if err != nil {
			return nil
		}
		syms = append(syms, fileSyms...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	rankSymbols(syms, goal)
	return syms, nil
}

func parseGoSymbols(absPath, rel string) ([]Symbol, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var out []Symbol
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name != nil {
			pos := fset.Position(fn.Pos())
			out = append(out, Symbol{Path: rel, Name: fn.Name.Name, Line: pos.Line})
		}
	}
	return out, nil
}

func rankSymbols(syms []Symbol, goal string) {
	goal = strings.ToLower(goal)
	sort.SliceStable(syms, func(i, j int) bool {
		return scoreSymbol(syms[i], goal) > scoreSymbol(syms[j], goal)
	})
}

func scoreSymbol(s Symbol, goal string) int {
	score := 0
	low := strings.ToLower(s.Path + " " + s.Name)
	for _, tok := range strings.Fields(goal) {
		if len(tok) < 3 {
			continue
		}
		if strings.Contains(low, strings.ToLower(tok)) {
			score += 10
		}
	}
	if strings.HasSuffix(s.Path, "_test.go") && strings.Contains(goal, "test") {
		score += 5
	}
	return score
}

// FormatSymbols renders symbols for the prompt (paths and names only).
func FormatSymbols(syms []Symbol, maxBytes int) string {
	var b strings.Builder
	for _, s := range syms {
		line := s.Path + ":" + s.Name + ":" + itoa(s.Line) + "\n"
		if maxBytes > 0 && b.Len()+len(line) > maxBytes {
			break
		}
		b.WriteString(line)
	}
	return b.String()
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
