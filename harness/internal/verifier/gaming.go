package verifier

import (
	"path/filepath"
	"strings"
)

// ProtectedTestPath reports whether a path is a protected test file.
func ProtectedTestPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	if strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.js") {
		return true
	}
	if strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.js") {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return true
	}
	return false
}

// NoteTestEdit records a protected test edit for the session summary.
func NoteTestEdit(session interface{ SetProtectedTestEdit() }, path string) {
	if ProtectedTestPath(path) {
		session.SetProtectedTestEdit()
	}
}
