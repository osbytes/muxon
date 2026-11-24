package toolcall

import (
	"errors"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

// ErrNoToolCalls means the model replied with text and no tool call.
var ErrNoToolCalls = errors.New("no tool calls in response")

// ErrMultipleToolCalls means the model returned more calls than this profile allows.
var ErrMultipleToolCalls = errors.New("expected a single tool call")

// MaxBatchCalls is the most native tool calls one reply may contain.
const MaxBatchCalls = 8

// ParseNative extracts tool calls from a provider response.
// When parallel is false, more than one call is an error. When it is true, up to MaxBatchCalls are accepted.
func ParseNative(resp provider.Response, parallel bool) (Outcome, error) {
	n := len(resp.ToolCalls)
	if n == 0 {
		return Outcome{Text: resp.Text}, ErrNoToolCalls
	}
	if n > 1 && (!parallel || n > MaxBatchCalls) {
		return Outcome{}, ErrMultipleToolCalls
	}
	return Outcome{Calls: resp.ToolCalls, Text: resp.Text}, nil
}
