package contextengine

import (
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func TestCompactDropsTokensKeepsVerifyFailure(t *testing.T) {
	e := NewEstimator()
	var msgs []provider.Message
	for i := 0; i < 20; i++ {
		msgs = append(msgs, provider.Message{
			Role:    "tool",
			Name:    "read",
			Content: strings.Repeat("x", 500),
		})
	}
	msgs = append(msgs, provider.Message{Role: "user", Content: "verify failed:\n" + strings.Repeat("fail", 100)})
	before := e.EstimateMessages(msgs)
	out := CompactHistory(msgs, before/3, 32768, e)
	after := e.EstimateMessages(out)
	if after >= before {
		t.Fatalf("expected drop before=%d after=%d", before, after)
	}
	found := false
	for _, m := range out {
		if strings.Contains(m.Content, verifyFailureMarker) {
			found = true
		}
	}
	if !found {
		t.Fatal("verify failure elided")
	}
}

func TestCompactKeepsRecentTailAndElidesOlderRead(t *testing.T) {
	e := NewEstimator()
	old := strings.Repeat("x", 8000)
	msgs := []provider.Message{{Role: "user", Content: "where is the loop?"}}
	msgs = append(msgs, toolRound("read", old)...)
	msgs = append(msgs, toolRound("read", old)...)
	msgs = append(msgs, toolRound("read", "KEEP_ONE")...)
	msgs = append(msgs, toolRound("search", "KEEP_TWO")...)

	out := CompactHistory(msgs, 4000, 32768, e)
	joined := joinContents(out)
	if strings.Contains(joined, old) {
		t.Fatalf("CompactHistory() kept an older read body")
	}
	if !strings.Contains(joined, "KEEP_ONE") || !strings.Contains(joined, "KEEP_TWO") {
		t.Fatalf("CompactHistory() recent tail = %q, want KEEP_ONE and KEEP_TWO", joined)
	}
	if strings.Contains(joined, "Earlier in this session:") {
		t.Fatalf("CompactHistory() folded while under budget")
	}
	if !strings.Contains(joined, "where is the loop?") {
		t.Fatalf("CompactHistory() dropped the user line")
	}
}

func TestCompactFoldsOnlyWhenStillOverBudget(t *testing.T) {
	e := NewEstimator()
	old := strings.Repeat("y", 8000)
	msgs := []provider.Message{{Role: "user", Content: "explain tool calling"}}
	msgs = append(msgs, toolRound("read", old)...)
	msgs = append(msgs, toolRound("read", old)...)
	msgs = append(msgs, toolRound("read", "KEEP_ONE")...)
	msgs = append(msgs, toolRound("search", "KEEP_TWO")...)

	out := CompactHistory(msgs, 15, 32768, e)
	joined := joinContents(out)
	if strings.Contains(joined, old) {
		t.Fatalf("CompactHistory() kept an older read body after fold")
	}
	if !strings.Contains(joined, "KEEP_ONE") || !strings.Contains(joined, "KEEP_TWO") {
		t.Fatalf("CompactHistory() folded the recent tail: %q", joined)
	}
	if !strings.Contains(joined, "Earlier in this session:") {
		t.Fatalf("CompactHistory() = %q, want a folded older span", joined)
	}
	if !strings.Contains(joined, "user: explain tool calling") || !strings.Contains(joined, "tool read") {
		t.Fatalf("CompactHistory() summary = %q, want the user line and tool name", joined)
	}
}

func toolRound(name, body string) []provider.Message {
	return []provider.Message{
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: name, Name: name, Arguments: []byte(`{}`)}}},
		{Role: "tool", Name: name, ToolCallID: name, Content: body},
	}
}

func joinContents(msgs []provider.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}
