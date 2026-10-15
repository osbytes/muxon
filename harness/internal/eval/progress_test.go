package eval

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestTrialProgressLines(t *testing.T) {
	var buf bytes.Buffer
	p := NewTrialProgress(&buf)
	p.SetTotal(2)
	p.Banner("run-1", 2)
	idx := p.Start("fix-test-001", "opencode-go-kimi", 1)
	p.Done(idx, ResultRow{
		TaskID:  "fix-test-001",
		ModelID: "opencode-go-kimi",
		Trial:   1,
		Reward:  0,
		WallS:   12.4,
		CostUSD: 0.021,
		Reason:  "timeout",
	})
	p.Done(2, ResultRow{
		TaskID:  "off-by-one-001",
		ModelID: "opencode-go-kimi",
		Trial:   1,
		Reward:  1,
		WallS:   8,
		CostUSD: 0.050,
	})
	p.Error(2, "off-by-one-001", "opencode-go-kimi", 1, errors.New("missing API key"))

	got := buf.String()
	for _, want := range []string{
		"run run-1  2 trials\n",
		"start  1/2  fix-test-001  opencode-go-kimi  trial 1\n",
		"fail  1/2  fix-test-001  opencode-go-kimi  trial 1  12.4s  $0.021  total $0.021  timeout\n",
		"pass  2/2  off-by-one-001  opencode-go-kimi  trial 1  8.0s  $0.050  total $0.071\n",
		"fail   2/2  off-by-one-001  opencode-go-kimi  trial 1  missing API key\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
