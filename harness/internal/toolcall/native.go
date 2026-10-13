package toolcall

import (
	"context"
	"fmt"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// Outcome is a parsed tool call from a model response.
type Outcome struct {
	Calls []provider.ToolCall
	Text  string
}

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

// Next completes one native tool-calling turn.
func Next(ctx context.Context, p provider.Provider, req provider.Request) (Outcome, provider.Response, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return Outcome{}, resp, err
	}
	out, err := ParseNative(resp)
	return out, resp, err
}
