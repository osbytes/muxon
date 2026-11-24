package patch

import (
	"fmt"
	"strings"
)

// ApplySearchReplace runs the patch ladder for SEARCH/REPLACE edits.
func ApplySearchReplace(content string, reps []Replacement) ([]byte, string, error) {
	out, strategy, err := ApplySearchReplaceExact(content, reps)
	if err == nil {
		return out, strategy, nil
	}
	out, strategy, err = applySearchReplaceWhitespace(content, reps)
	if err == nil {
		return out, strategy, nil
	}
	out, strategy, err = applySearchReplaceFuzzy(content, reps)
	if err == nil {
		return out, strategy, nil
	}
	return nil, "", rejectWithContext(content, reps)
}

func applySearchReplaceWhitespace(content string, reps []Replacement) ([]byte, string, error) {
	norm := normalizeEOL(content)
	out := norm
	for _, r := range reps {
		search := normalizeEOL(r.Search)
		replace := normalizeEOL(r.Replace)
		if search == "" {
			return nil, "", fmt.Errorf("empty search block")
		}
		if countNormalized(out, search) != 1 {
			return nil, "", fmt.Errorf("whitespace: not unique")
		}
		out = replaceNormalizedOnce(out, search, replace)
	}
	return []byte(out), "whitespace", nil
}

func applySearchReplaceFuzzy(content string, reps []Replacement) ([]byte, string, error) {
	out := content
	for _, r := range reps {
		if r.Search == "" {
			return nil, "", fmt.Errorf("empty search block")
		}
		idx, score := uniqueFuzzyMatch(out, r.Search)
		if idx < 0 || score < 0.85 {
			return nil, "", fmt.Errorf("fuzzy: no unique match")
		}
		out = out[:idx] + r.Replace + out[idx+len(r.Search):]
	}
	return []byte(out), "fuzzy", nil
}

func rejectWithContext(content string, reps []Replacement) error {
	if len(reps) == 0 {
		return fmt.Errorf("patch rejected")
	}
	search := reps[0].Search
	idx := strings.Index(content, search)
	if idx < 0 {
		idx = 0
	}
	start := max(0, idx-40)
	end := min(len(content), idx+len(search)+40)
	snippet := content[start:end]
	return fmt.Errorf("patch rejected; nearest lines:\n%s", snippet)
}

func normalizeEOL(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

func countNormalized(content, search string) int {
	nc := normalizeEOL(content)
	ns := normalizeEOL(search)
	return strings.Count(nc, ns)
}

func replaceNormalizedOnce(content, search, replace string) string {
	nc := normalizeEOL(content)
	ns := normalizeEOL(search)
	nr := normalizeEOL(replace)
	i := strings.Index(nc, ns)
	if i < 0 {
		return content
	}
	return nc[:i] + nr + nc[i+len(ns):]
}

func uniqueFuzzyMatch(content, search string) (int, float64) {
	if strings.Count(content, search) == 1 {
		return strings.Index(content, search), 1.0
	}
	bestIdx := -1
	bestScore := 0.0
	count := 0
	for i := 0; i+len(search) <= len(content); i++ {
		window := content[i : i+len(search)]
		sc := similarity(window, search)
		if sc >= 0.85 {
			if sc > bestScore {
				bestScore = sc
				bestIdx = i
				count = 1
			} else if sc == bestScore && i != bestIdx {
				count++
			}
		}
	}
	if count != 1 {
		return -1, 0
	}
	return bestIdx, bestScore
}

func similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	na := normalizeEOL(a)
	nb := normalizeEOL(b)
	if na == nb {
		return 1
	}
	if len(na) == 0 || len(nb) == 0 {
		return 0
	}
	matches := 0
	for i := 0; i < len(na) && i < len(nb); i++ {
		if na[i] == nb[i] {
			matches++
		}
	}
	denom := len(na)
	if len(nb) > denom {
		denom = len(nb)
	}
	return float64(matches) / float64(denom)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ApplyUDiff runs the patch ladder for unified diffs.
func ApplyUDiff(content string, diff string) ([]byte, string, error) {
	out, strategy, err := ApplyUDiffExact(content, diff)
	if err == nil {
		return out, strategy, nil
	}
	return nil, "", fmt.Errorf("patch rejected: %v", err)
}
