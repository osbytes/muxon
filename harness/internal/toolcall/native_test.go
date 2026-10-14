package toolcall

import (
	"errors"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func calls(n int) provider.Response {
	out := provider.Response{FinishReason: "tool_calls"}
	for i := 0; i < n; i++ {
		out.ToolCalls = append(out.ToolCalls, provider.ToolCall{ID: string(rune('a' + i)), Name: "read"})
	}
	return out
}

func TestParseNativeBatch(t *testing.T) {
	got, err := ParseNative(calls(3), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(got.Calls))
	}
	if _, err := ParseNative(calls(3), false); !errors.Is(err, ErrMultipleToolCalls) {
		t.Fatalf("parallel off err = %v, want ErrMultipleToolCalls", err)
	}
	if _, err := ParseNative(calls(MaxBatchCalls+1), true); !errors.Is(err, ErrMultipleToolCalls) {
		t.Fatalf("over cap err = %v, want ErrMultipleToolCalls", err)
	}
}
