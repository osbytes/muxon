package toolcall

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type seqProvider struct {
	resps []provider.Response
	i     int
}

func (s *seqProvider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	if s.i >= len(s.resps) {
		return provider.Response{}, context.Canceled
	}
	r := s.resps[s.i]
	s.i++
	return r, nil
}

func (s *seqProvider) Stream(ctx context.Context, req provider.Request) (provider.Stream, error) {
	return nil, context.Canceled
}

func TestTwoStepYieldsOneCall(t *testing.T) {
	args, _ := json.Marshal(map[string]interface{}{"path": "a.go", "start_line": 1, "end_line": 2})
	p := &seqProvider{resps: []provider.Response{
		{Text: `{"tool":"read"}`},
		{Text: string(args)},
	}}
	specs := []tools.Spec{{Name: "read", Parameters: schemaParams()}}
	req := provider.Request{ModelID: "m", Messages: []provider.Message{{Role: "user", Content: "go"}}}
	out, _, err := NextTwoStep(context.Background(), p, req, TwoStepInput{Phase: tools.PhaseEditing, Specs: specs})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Calls) != 1 || out.Calls[0].Name != "read" {
		t.Fatalf("%+v", out.Calls)
	}
}

func schemaParams() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"start_line":{"type":"integer"},"end_line":{"type":"integer"}},"required":["path","start_line","end_line"]}`)
}
