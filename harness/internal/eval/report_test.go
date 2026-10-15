package eval

import (
	"strings"
	"testing"
)

func TestFormatReport(t *testing.T) {
	text := FormatReport([]ResultRow{{
		TaskID: "fix-test-001", Reward: 1, CostUSD: 0.1, InputTokens: 10, OutputTokens: 2, WallS: 1,
		ToolRounds: 2, SalvagedToolCalls: 1,
	}})
	if !strings.Contains(text, "pass rate        1.000") {
		t.Fatalf("%s", text)
	}
	if !strings.Contains(text, "salvage          0.500") {
		t.Fatalf("%s", text)
	}
	if !strings.Contains(text, "total cost       $0.100") {
		t.Fatalf("%s", text)
	}
}

func TestCompareRowsMcNemar(t *testing.T) {
	a := []ResultRow{{TaskID: "a", ModelID: "m", Reward: 1}, {TaskID: "b", ModelID: "m", Reward: 0}}
	b := []ResultRow{{TaskID: "a", ModelID: "m", Reward: 0}, {TaskID: "b", ModelID: "m", Reward: 1}}
	text := CompareRows(a, b)
	if !strings.Contains(text, "a_only_pass 1") || !strings.Contains(text, "b_only_pass 1") {
		t.Fatalf("%s", text)
	}
	if !strings.Contains(text, "mcnemar_p 1.0000") {
		t.Fatalf("%s", text)
	}
}

func TestBootstrapPassRatePinned(t *testing.T) {
	rows := []ResultRow{
		{TaskID: "a", Reward: 1},
		{TaskID: "b", Reward: 0},
	}
	lo, hi := BootstrapPassRate(rows, 200, 1)
	if lo < 0 || hi > 1 || lo > hi {
		t.Fatalf("ci %v %v", lo, hi)
	}
}
