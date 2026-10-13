package toolcall

import (
	"fmt"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// ParseNative extracts tool calls from a provider response.
func ParseNative(resp provider.Response) (Outcome, error) {
	if len(resp.ToolCalls) == 0 {
		return Outcome{Text: resp.Text}, fmt.Errorf("no tool calls in response")
	}
	if len(resp.ToolCalls) > 1 {
		return Outcome{}, fmt.Errorf("expected single tool call in M0")
	}
	return Outcome{Calls: resp.ToolCalls, Text: resp.Text}, nil
}

