package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type finishTool struct{}

// NewFinish returns the finish tool.
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
		return tools.Fail(env, "finish", "invalid finish args"), nil
	}
	if env.Session == nil {
		return tools.Fail(env, "finish", "no session"), nil
	}
	if len(env.Session.OpenTodos) > 0 {
		return tools.Fail(env, "finish", "open todo items remain"), nil
	}
	env.Session.FinishSummary = a.Summary
	if env.Workspace.Changed() {
		env.Session.PendingVerify = true
		return tools.OK(env, "finish", "pending verify", 0, 0), nil
	}
	return tools.OK(env, "finish", "done", 0, 0), nil
}
