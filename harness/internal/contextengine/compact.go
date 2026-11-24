package contextengine

import (
	"strings"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

const verifyFailureMarker = "verify failed"

// CompactHistory elides tool results outside the recent tail once history
// passes half of budgetTokens, and folds that older span when the remainder
// is still over the budget. The recent tail and verify failures stay verbatim.
func CompactHistory(msgs []provider.Message, budgetTokens, contextWindow int, est *Estimator) []provider.Message {
	if budgetTokens <= 0 || len(msgs) == 0 {
		return msgs
	}
	if est == nil {
		est = NewEstimator()
	}
	if est.EstimateMessages(msgs) <= budgetTokens/2 {
		return msgs
	}
	tail := recentTailStart(msgs, recentToolRounds(contextWindow))
	out := append([]provider.Message(nil), msgs...)
	seen := make(map[string]int)
	for i := 0; i < tail && i < len(out); i++ {
		if out[i].Role != "tool" || isVerifyFailure(out[i]) {
			continue
		}
		key := out[i].Name + "|" + hashContent(out[i].Content)
		seen[key]++
		body := out[i].Content
		if seen[key] > 1 {
			body = "[elided duplicate tool result]"
		} else if len(body) > 2000 {
			body = body[:2000] + "\n...[truncated]"
		}
		out[i].Content = body
	}
	if est.EstimateMessages(out) > budgetTokens/2 {
		for i := 0; i < tail && i < len(out); i++ {
			if out[i].Role != "tool" || isVerifyFailure(out[i]) {
				continue
			}
			out[i].Content = "[elided tool output]"
		}
	}
	if est.EstimateMessages(out) > budgetTokens {
		return foldOlder(out, tail)
	}
	return out
}

func recentToolRounds(contextWindow int) int {
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	switch {
	case contextWindow <= 32768:
		return 2
	case contextWindow <= 65536:
		return 4
	default:
		return 8
	}
}

func toolRoundStarts(msgs []provider.Message) []int {
	var starts []int
	openedByAssistant := false
	for i, m := range msgs {
		if len(m.ToolCalls) > 0 {
			starts = append(starts, i)
			openedByAssistant = true
			continue
		}
		if m.Role == "tool" {
			if !openedByAssistant {
				starts = append(starts, i)
			}
			continue
		}
		openedByAssistant = false
	}
	return starts
}

func recentTailStart(msgs []provider.Message, rounds int) int {
	if rounds < 1 {
		rounds = 1
	}
	starts := toolRoundStarts(msgs)
	if len(starts) <= rounds {
		return 0
	}
	return starts[len(starts)-rounds]
}

func isVerifyFailure(m provider.Message) bool {
	return strings.Contains(m.Content, verifyFailureMarker)
}

func foldOlder(msgs []provider.Message, tail int) []provider.Message {
	if tail <= 0 {
		return msgs
	}
	var b strings.Builder
	b.WriteString("Earlier in this session:")
	wrote := false
	var kept []provider.Message
	for _, m := range msgs[:tail] {
		if isVerifyFailure(m) {
			kept = append(kept, m)
			continue
		}
		for _, line := range foldLines(m) {
			b.WriteString("\n- ")
			b.WriteString(line)
			wrote = true
		}
	}
	out := make([]provider.Message, 0, 1+len(kept)+len(msgs)-tail)
	if wrote {
		out = append(out, provider.Message{Role: "user", Content: b.String()})
	}
	out = append(out, kept...)
	out = append(out, msgs[tail:]...)
	return out
}

func foldLines(m provider.Message) []string {
	var lines []string
	if m.Role == "user" || m.Role == "assistant" {
		if text := firstLine(m.Content); text != "" {
			lines = append(lines, m.Role+": "+text)
		}
	}
	if m.Role == "tool" && m.Name != "" {
		lines = append(lines, "tool "+m.Name)
	}
	for _, call := range m.ToolCalls {
		if call.Name != "" {
			lines = append(lines, "tool "+call.Name)
		}
	}
	return lines
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	if len(text) > 160 {
		text = text[:160]
	}
	return text
}

func hashContent(s string) string {
	if len(s) > 64 {
		return s[:64]
	}
	return s
}
