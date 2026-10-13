package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type finishTool struct{}

func NewFinish() tools.Tool { return &finishTool{} }

func (finishTool) Name() string { return "finish" }

func (finishTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "finish",
		Description: "Mark the goal complete",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`),
	}
}

type finishArgs struct {
	Summary string `json:"summary"`
}

func (finishTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	var a finishArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Result{OK: false, Text: "invalid finish args"}, nil
	}
	if env.Session == nil {
		return tools.Result{OK: false, Text: "no session"}, nil
	}
	env.Session.FinishSummary = a.Summary
	if env.Workspace.Changed() {
		env.Session.PendingVerify = true
		return tools.Result{OK: true, Text: "pending verify"}, nil
	}
	return tools.Result{OK: true, Text: "done"}, nil
}
