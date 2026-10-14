package toolcall

import (
	"errors"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// ErrNoToolCalls means the model replied with text and no tool call.
var ErrNoToolCalls = errors.New("no tool calls in response")

// ErrMultipleToolCalls means the model returned more than one tool call.
var ErrMultipleToolCalls = errors.New("expected a single tool call")

// ParseNative extracts tool calls from a provider response.
func ParseNative(resp provider.Response) (Outcome, error) {
	if len(resp.ToolCalls) == 0 {
		return Outcome{Text: resp.Text}, ErrNoToolCalls
	}
	if len(resp.ToolCalls) > 1 {
		return Outcome{}, ErrMultipleToolCalls
	}
	return Outcome{Calls: resp.ToolCalls, Text: resp.Text}, nil
}
