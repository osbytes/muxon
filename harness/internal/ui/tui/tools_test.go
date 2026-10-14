package tui

import "testing"

func TestFailedToolSummaryShowsInnerError(t *testing.T) {
	tc := &toolCall{
		name: "read",
		done: true,
		output: `<tool_result tool="read" trust="workspace">
open /tmp/missing.go: no such file or directory
</tool_result>`,
	}
	got := tc.summary()
	want := "open /tmp/missing.go: no such file or directory"
	if got != want {
		t.Fatalf("summary() = %q, want %q", got, want)
	}
}
