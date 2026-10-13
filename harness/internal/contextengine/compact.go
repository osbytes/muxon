package contextengine

import (
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

const verifyFailureMarker = "verify failed"

// CompactHistory elides and truncates tool history when over budget.
func CompactHistory(msgs []provider.Message, budgetTokens int, est *Estimator) []provider.Message {
	if budgetTokens <= 0 {
		return msgs
	}
	if est.EstimateMessages(msgs) <= budgetTokens {
		return msgs
	}
	out := make([]provider.Message, 0, len(msgs))
	seen := make(map[string]int)
	verifyIdx := -1
	for i, m := range msgs {
		if strings.Contains(m.Content, verifyFailureMarker) {
			verifyIdx = i
		}
	}
	for i, m := range msgs {
		if i == verifyIdx {
			out = append(out, m)
			continue
		}
		if m.Role == "tool" {
			key := m.Name + "|" + hashContent(m.Content)
			seen[key]++
			if seen[key] > 1 {
				out = append(out, provider.Message{
					Role:       m.Role,
					Name:       m.Name,
					ToolCallID: m.ToolCallID,
					Content:    "[elided duplicate tool result]",
				})
				continue
			}
			body := m.Content
			if len(body) > 2000 {
				body = body[:2000] + "\n...[truncated]"
			}
			out = append(out, provider.Message{
				Role:       m.Role,
				Name:       m.Name,
				ToolCallID: m.ToolCallID,
				Content:    body,
			})
			continue
		}
		out = append(out, m)
	}
	if est.EstimateMessages(out) > budgetTokens {
		return truncateOldestToolResults(out, budgetTokens, est, verifyIdx)
	}
	return out
}

func truncateOldestToolResults(msgs []provider.Message, budget int, est *Estimator, protectIdx int) []provider.Message {
	out := append([]provider.Message(nil), msgs...)
	for i := 0; i < len(out) && est.EstimateMessages(out) > budget; i++ {
		if i == protectIdx {
			continue
		}
		if out[i].Role != "tool" {
			continue
		}
		if strings.Contains(out[i].Content, verifyFailureMarker) {
			continue
		}
		out[i].Content = "[elided tool output]"
	}
	return out
}

func hashContent(s string) string {
	if len(s) > 64 {
		return s[:64]
	}
	return s
}
