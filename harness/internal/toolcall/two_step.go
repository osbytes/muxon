package toolcall

import (
	"context"
	"fmt"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

// TwoStepInput drives a name-then-args tool turn.
type TwoStepInput struct {
	Protocol string
	Phase    string
	Specs    []tools.Spec
}

// NextTwoStep runs two-step tool calling.
func NextTwoStep(ctx context.Context, p provider.Provider, req provider.Request, in TwoStepInput) (Outcome, provider.Response, error) {
	allowed := activeSpecs(in.Phase, in.Specs)
	names := make([]string, 0, len(allowed))
	nameToSpec := make(map[string]tools.Spec)
	for _, s := range allowed {
		names = append(names, s.Name)
		nameToSpec[s.Name] = s
	}
	req1 := req
	req1.Tools = nil
	req1.ResponseFormat = Step1Schema(names)
	resp1, err := completeWithConstraintRetry(ctx, p, req1)
	if err != nil {
		return Outcome{}, resp1, err
	}
	toolName, err := pickToolName(resp1, nameToSpec)
	if err != nil {
		return Outcome{}, resp1, err
	}
	spec := nameToSpec[toolName]
	req2 := req
	req2.Messages = append(req2.Messages, provider.Message{Role: "assistant", Content: resp1.Text})
	req2.Messages = append(req2.Messages, provider.Message{
		Role:    "user",
		Content: "Fill arguments for tool " + toolName + " as JSON only.",
	})
	req2.ResponseFormat = Step2Schema(spec.Parameters)
	resp2, err := completeWithConstraintRetry(ctx, p, req2)
	if err != nil {
		return Outcome{}, resp2, err
	}
	salv, err := SalvageText(resp2.Text, toolName, spec)
	if err == nil {
		return Outcome{Calls: []provider.ToolCall{salv.Call}, Text: resp2.Text, Salvaged: salv.Salvaged}, resp2, nil
	}
	if len(resp2.ToolCalls) > 0 {
		return Outcome{Calls: resp2.ToolCalls, Text: resp2.Text}, resp2, nil
	}
	return Outcome{}, resp2, fmt.Errorf("two_step step2: %w", err)
}

func pickToolName(resp provider.Response, allowed map[string]tools.Spec) (string, error) {
	if len(resp.ToolCalls) == 1 {
		return resp.ToolCalls[0].Name, nil
	}
	name, _, err := SalvageStep1(resp.Text, allowed)
	return name, err
}

func activeSpecs(phase string, specs []tools.Spec) []tools.Spec {
	var out []tools.Spec
	for _, s := range specs {
		if err := tools.CheckPhase(phase, s.Name); err == nil {
			out = append(out, s)
		}
	}
	return out
}

func completeWithConstraintRetry(ctx context.Context, p provider.Provider, req provider.Request) (provider.Response, error) {
	resp, err := p.Complete(ctx, req)
	if err == nil {
		return resp, nil
	}
	if api, ok := err.(*provider.APIError); ok && api.Kind == provider.APIErrorConstraintRejected {
		req.ResponseFormat = DowngradeFormat(req.ResponseFormat)
		return p.Complete(ctx, req)
	}
	return resp, err
}
