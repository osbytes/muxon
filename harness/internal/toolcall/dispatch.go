package toolcall

import (
	"context"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

// DispatchInput selects the tool-calling protocol.
type DispatchInput struct {
	ToolProtocol string
	Phase        string
	Specs        []tools.Spec
	Parallel     bool
}

// Outcome is a parsed tool call from a model response.
type Outcome struct {
	Calls    []provider.ToolCall
	Text     string
	Salvaged bool
}

// Next completes one tool-calling turn for the model protocol.
func Next(ctx context.Context, p provider.Provider, req provider.Request, in DispatchInput) (Outcome, provider.Response, error) {
	switch in.ToolProtocol {
	case "two_step":
		o, r, err := NextTwoStep(ctx, p, req, TwoStepInput{Phase: in.Phase, Specs: in.Specs})
		return Outcome{Calls: o.Calls, Text: o.Text, Salvaged: o.Salvaged}, r, err
	case "fenced":
		o, r, err := NextFenced(ctx, p, req, TwoStepInput{Phase: in.Phase, Specs: in.Specs})
		return Outcome{Calls: o.Calls, Text: o.Text, Salvaged: o.Salvaged}, r, err
	default:
		return nextNative(ctx, p, req, in.Parallel)
	}
}

func nextNative(ctx context.Context, p provider.Provider, req provider.Request, parallel bool) (Outcome, provider.Response, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return Outcome{}, resp, err
	}
	text := StripReasoning(resp.Text)
	resp.Text = text
	out, err := ParseNative(resp, parallel)
	if err != nil && text != "" {
		return Outcome{Text: text}, resp, err
	}
	return Outcome{Calls: out.Calls, Text: out.Text}, resp, err
}
