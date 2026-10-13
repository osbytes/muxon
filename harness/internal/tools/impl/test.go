package impl

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/subproc"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type testTool struct{}

func NewTest() tools.Tool { return &testTool{} }

func (testTool) Name() string { return "test" }

func (testTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "test",
		Description: "Run one filtered test. Does not mark the goal verified.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"filter":{"type":"string"}},"required":["filter"]}`),
	}
}

type testArgs struct {
	Filter string `json:"filter"`
}

func (testTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	if env == nil {
		return tools.Result{OK: false, Text: "workspace is not trusted"}, nil
	}
	if !env.Trusted {
		return tools.Fail(env, "test", "workspace is not trusted"), nil
	}
	if env.FilterCommand == "" || !strings.Contains(env.FilterCommand, "{filter}") {
		return tools.Fail(env, "test", "filter_command is not configured"), nil
	}
	var a testArgs
	if err := json.Unmarshal(args, &a); err != nil || strings.TrimSpace(a.Filter) == "" {
		return tools.Fail(env, "test", "filter is required"), nil
	}
	if strings.ContainsAny(a.Filter, " \t\n") {
		return tools.Fail(env, "test", "filter must be one token"), nil
	}
	fields := strings.Fields(env.FilterCommand)
	argv := make([]string, len(fields))
	for i, f := range fields {
		argv[i] = strings.ReplaceAll(f, "{filter}", a.Filter)
	}
	out, code, err := subproc.Run(ctx, env.Workspace.Root(), argv, nil)
	if err != nil && code == 0 {
		return tools.Fail(env, "test", err.Error()), nil
	}
	text := strings.TrimSpace(string(out))
	if code != 0 {
		return tools.Fail(env, "test", text), nil
	}
	return tools.OK(env, "test", text, 0, 0), nil
}
