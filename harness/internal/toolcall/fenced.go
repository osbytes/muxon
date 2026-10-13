package toolcall

import (
	"context"
	"fmt"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

// NextFenced uses two-step flow with fenced-block salvage on step 2.
func NextFenced(ctx context.Context, p provider.Provider, req provider.Request, in TwoStepInput) (Outcome, provider.Response, error) {
	out, resp, err := NextTwoStep(ctx, p, req, in)
	if err != nil {
		if len(resp.Text) == 0 {
			return out, resp, err
		}
		allowed := activeSpecs(in.Phase, in.Specs)
		for name, spec := range mapFromSpecs(allowed) {
			salv, serr := SalvageText(resp.Text, name, spec)
			if serr == nil {
				return Outcome{Calls: []provider.ToolCall{salv.Call}, Salvaged: true}, resp, nil
			}
		}
		return out, resp, fmt.Errorf("fenced: %w", err)
	}
	return out, resp, nil
}

func mapFromSpecs(specs []tools.Spec) map[string]tools.Spec {
	m := make(map[string]tools.Spec)
	for _, s := range specs {
		m[s.Name] = s
	}
	return m
}
