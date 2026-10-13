package security

import (
	"regexp"
	"strings"
)

var (
	skPattern    = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)
	apiKeyAssign = regexp.MustCompile(`(?i)([A-Z0-9_]*API_KEY)\s*=\s*[^\s]+`)
)

// RedactSecrets replaces likely secrets in text.
func RedactSecrets(s string) string {
	s = skPattern.ReplaceAllString(s, "sk-[REDACTED]")
	s = apiKeyAssign.ReplaceAllString(s, "$1=[REDACTED]")
	return s
}

// RedactLines applies RedactSecrets per line.
func RedactLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = RedactSecrets(l)
	}
	return out
}

// TruncateBytes caps output length with a suffix note.
func TruncateBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
}

// TruncateLines caps line count.
func TruncateLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n") + "\n...[truncated]"
}
