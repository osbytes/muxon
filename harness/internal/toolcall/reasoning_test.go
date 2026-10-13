package toolcall

import "testing"

func TestStripReasoning(t *testing.T) {
	in := thinkOpen + "secret" + thinkClose + "\n{\"tool\":\"read\"}"
	out := StripReasoning(in)
	if out != "{\"tool\":\"read\"}" {
		t.Fatalf("got %q", out)
	}
}

func TestStripReasoningUnclosed(t *testing.T) {
	in := "ok " + thinkOpen + "thinking"
	out := StripReasoning(in)
	if out != "ok" {
		t.Fatalf("got %q", out)
	}
}
