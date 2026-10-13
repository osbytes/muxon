package patch

import (
	"fmt"
	"strings"
)

// Replacement is one SEARCH/REPLACE pair.
type Replacement struct {
	Search  string `json:"search"`
	Replace string `json:"replace"`
}

// ApplySearchReplaceExact applies exact unique replacements in order.
func ApplySearchReplaceExact(content string, reps []Replacement) ([]byte, string, error) {
	out := content
	for _, r := range reps {
		if r.Search == "" {
			return nil, "", fmt.Errorf("empty search block")
		}
		if strings.Count(out, r.Search) != 1 {
			return nil, "", fmt.Errorf("search block must match exactly once")
		}
		out = strings.Replace(out, r.Search, r.Replace, 1)
	}
	return []byte(out), "exact", nil
}

// ApplyUDiffExact applies a minimal unified diff with one hunk (exact lines).
func ApplyUDiffExact(content string, diff string) ([]byte, string, error) {
	lines := strings.Split(content, "\n")
	hunk, err := parseSingleHunk(diff)
	if err != nil {
		return nil, "", err
	}
	if hunk.oldStart < 1 || hunk.oldStart > len(lines)+1 {
		return nil, "", fmt.Errorf("hunk start out of range")
	}
	idx := hunk.oldStart - 1
	for _, hl := range hunk.lines {
		switch hl.kind {
		case ' ':
			if idx >= len(lines) || lines[idx] != hl.text {
				return nil, "", fmt.Errorf("context mismatch at line %d", idx+1)
			}
			idx++
		case '-':
			if idx >= len(lines) || lines[idx] != hl.text {
				return nil, "", fmt.Errorf("delete mismatch at line %d", idx+1)
			}
			lines = append(lines[:idx], lines[idx+1:]...)
		case '+':
			lines = append(lines[:idx], append([]string{hl.text}, lines[idx:]...)...)
			idx++
		default:
			return nil, "", fmt.Errorf("invalid hunk line")
		}
	}
	return []byte(strings.Join(lines, "\n")), "exact", nil
}

type hunkLine struct {
	kind byte
	text string
}

type singleHunk struct {
	oldStart int
	lines    []hunkLine
}

func parseSingleHunk(diff string) (singleHunk, error) {
	var h singleHunk
	var inHunk bool
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			// @@ -start,count +start,count @@
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return singleHunk{}, fmt.Errorf("bad hunk header")
			}
			old := strings.TrimPrefix(parts[1], "-")
			oldStart := strings.Split(old, ",")[0]
			n, err := parseInt(oldStart)
			if err != nil {
				return singleHunk{}, err
			}
			h.oldStart = n
			continue
		}
		if !inHunk {
			continue
		}
		if line == "" {
			continue
		}
		if line[0] != ' ' && line[0] != '+' && line[0] != '-' {
			continue
		}
		h.lines = append(h.lines, hunkLine{kind: line[0], text: line[1:]})
	}
	if !inHunk || len(h.lines) == 0 {
		return singleHunk{}, fmt.Errorf("no hunk found")
	}
	return h, nil
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("bad number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
