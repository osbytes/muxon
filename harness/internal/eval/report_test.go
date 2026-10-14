package eval

import (
	"strings"
	"testing"
)

func TestFormatReport(t *testing.T) {
	text := FormatReport([]ResultRow{{
		TaskID: "fix-test-001", Reward: 1, CostUSD: 0.1, InputTokens: 10, OutputTokens: 2, WallS: 1,
	}})
	if !strings.Contains(text, "pass_rate 1.000") {
		t.Fatalf("%s", text)
	}
}
