package contextengine

import (
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func TestEstimatorCorrectionRatio(t *testing.T) {
	e := NewEstimator()
	msgs := []provider.Message{{Role: "user", Content: stringsRepeat("a", 35)}}
	est := e.EstimateMessages(msgs)
	if est < 8 || est > 12 {
		t.Fatalf("raw approx %d", est)
	}
	e.ObserveUsage(est*2, est)
	est2 := e.EstimateMessages(msgs)
	if est2 < est*2-2 || est2 > est*2+2 {
		t.Fatalf("expected ~2x, got %d from %d", est2, est)
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = s[0]
	}
	return string(b)
}
