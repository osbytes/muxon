package director

import (
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func TestRepeatedCallAfterEarlierWrite(t *testing.T) {
	d := &loopDetector{}
	write := []byte(`{"path":"a.go","content":"package a"}`)
	d.RecordToolCall("write", write, 0)
	d.RecordResult(provider.ToolCall{Name: "write", Arguments: write}, true)
	read := []byte(`{"path":"b.go","start_line":1,"end_line":4}`)
	for range 3 {
		d.RecordToolCall("read", read, 1)
		d.RecordResult(provider.ToolCall{Name: "read", Arguments: read}, true)
	}
	_, pattern, fire := d.Check()
	if !fire || pattern != "repeated_call" {
		t.Fatalf("fire=%v pattern=%q, want repeated_call", fire, pattern)
	}
}

func TestRepeatedCallResetsWhenRevisionChanges(t *testing.T) {
	d := &loopDetector{}
	read := []byte(`{"path":"b.go","start_line":1,"end_line":4}`)
	d.RecordToolCall("read", read, 0)
	d.RecordToolCall("read", read, 0)
	d.RecordToolCall("write", []byte(`{"path":"a.go"}`), 0)
	d.RecordToolCall("read", read, 1)
	if _, _, fire := d.Check(); fire {
		t.Fatal("repeat fired across a revision change")
	}
}

func TestRereadFiresOnThirdOverlap(t *testing.T) {
	d := &loopDetector{}
	first := []byte(`{"path":"a.go","start_line":1,"end_line":10}`)
	second := []byte(`{"path":"a.go","start_line":4,"end_line":12}`)
	third := []byte(`{"path":"a.go","start_line":8,"end_line":20}`)
	d.RecordToolCall("read", first, 0)
	d.RecordToolCall("read", second, 0)
	if _, _, fire := d.Check(); fire {
		t.Fatal("re-read fired after two reads")
	}
	d.RecordToolCall("read", third, 0)
	_, pattern, fire := d.Check()
	if !fire || pattern != "reread" {
		t.Fatalf("fire=%v pattern=%q, want reread", fire, pattern)
	}
}

func TestPatchThrashFiresOnSecondRejection(t *testing.T) {
	d := &loopDetector{}
	args := []byte(`{"path":"a.go","replacements":[{"search":"old line","replace":"new line"}]}`)
	call := provider.ToolCall{Name: "patch", Arguments: args}
	d.RecordToolCall("patch", args, 0)
	d.RecordResult(call, false)
	if _, _, fire := d.Check(); fire {
		t.Fatal("patch thrash fired after one rejection")
	}
	d.RecordToolCall("patch", args, 1)
	d.RecordResult(call, false)
	_, pattern, fire := d.Check()
	if !fire || pattern != "patch_thrash" {
		t.Fatalf("fire=%v pattern=%q, want patch_thrash", fire, pattern)
	}

	other := &loopDetector{}
	a := []byte(`{"path":"a.go","replacements":[{"search":"one","replace":"x"}]}`)
	b := []byte(`{"path":"a.go","replacements":[{"search":"two","replace":"y"}]}`)
	other.RecordToolCall("patch", a, 0)
	other.RecordResult(provider.ToolCall{Name: "patch", Arguments: a}, false)
	other.RecordToolCall("patch", b, 0)
	other.RecordResult(provider.ToolCall{Name: "patch", Arguments: b}, false)
	if _, _, fire := other.Check(); fire {
		t.Fatal("different old text counted as the same rejection")
	}
}
