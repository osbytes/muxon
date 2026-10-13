package headless

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/observability"
)

func TestNDJSONIncludesSessionEnd(t *testing.T) {
	var buf bytes.Buffer
	p := &Presenter{Mode: ModeJSON, Out: &buf}
	p.Handle(observability.StateTransitionEvent("acting", "tooling", ""))
	p.Handle(observability.SessionEndEvent("done", "verified", observability.UsageTotals{}, 1, 1, 0, 0, 0))
	if !strings.Contains(buf.String(), `"type":"session.end"`) {
		t.Fatalf("%s", buf.String())
	}
}
