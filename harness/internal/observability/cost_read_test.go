package observability

import (
	"path/filepath"
	"testing"
)

func TestReadSessionCostFromTrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Append(SessionEndEvent("done", "verified", UsageTotals{
		InputTokens:  1000,
		OutputTokens: 50,
		CachedTokens: 800,
		CostUSD:      0.42,
	}, 3, 1, 0))
	w.Close()
	sum, _, err := ReadSessionCost(path)
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalCostUSD != 0.42 || sum.CachedTokens != 800 {
		t.Fatalf("%+v", sum)
	}
}
