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
	out := CompactHistory(msgs, before/3, e)
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
