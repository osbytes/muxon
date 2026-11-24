package security

import (
	"path/filepath"
	"strings"
)

// DenyReadMessage is returned when a path matches the secret deny list.
const DenyReadMessage = "path is on the secret deny list"

// Policy holds secret-path rules for read/search.
type Policy struct {
	extraDeny []string
}

// NewPolicy returns the built-in deny policy.
func NewPolicy() *Policy {
	return &Policy{}
}

// WithDenyRead adds user-configured deny patterns (from muxon.toml later).
func (p *Policy) WithDenyRead(patterns []string) *Policy {
	p.extraDeny = append(p.extraDeny, patterns...)
	return p
}

// DenyRead reports whether file contents must not be returned.
func (p *Policy) DenyRead(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == "" {
		return false
	}
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".env") {
		switch base {
		case ".env.example", ".env.sample", ".env.template":
			return false
		default:
			return true
		}
	}
	lower := strings.ToLower(rel)
	suffixDeny := []string{".pem", ".key", ".p12", ".pfx", ".keystore"}
	for _, suf := range suffixDeny {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	if strings.HasPrefix(base, "id_rsa") || strings.HasPrefix(base, "id_ed25519") {
		return true
	}
	prefixDeny := []string{
		".aws/", ".ssh/", ".gnupg/",
	}
	for _, pre := range prefixDeny {
		if strings.Contains(lower, pre) {
			return true
		}
	}
	if base == ".netrc" || base == ".npmrc" || base == ".pypirc" {
		return true
	}
	if strings.Contains(lower, "credentials") || strings.Contains(lower, "secrets.") {
		return true
	}
	if strings.HasSuffix(lower, ".tfstate") {
		return true
	}
	for _, pat := range p.extraDeny {
		if matchGlob(pat, rel) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	ok, _ := filepath.Match(pattern, path)
	if ok {
		return true
	}
	if strings.Contains(pattern, "**") {
		pattern = strings.ReplaceAll(pattern, "**", "*")
		ok, _ = filepath.Match(pattern, path)
	}
	return ok
}
