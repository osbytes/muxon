package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

const maxWriteBytes = 32 * 1024

type writeTool struct{}

func NewWrite() tools.Tool { return &writeTool{} }

func (writeTool) Name() string { return "write" }

func (writeTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "write",
		Description: "Write a file",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
	}
}

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (writeTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	_ = env.EnsureSecurity()
	var a writeArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "write", "invalid write args"), nil
	}
	if len(a.Content) > maxWriteBytes {
		return tools.Fail(env, "write", "content exceeds 32KiB cap"), nil
	}
	if err := env.Workspace.CheckFresh(a.Path); err != nil {
		return tools.Fail(env, "write", err.Error()), nil
	}
	if err := env.Workspace.WriteFile(a.Path, []byte(a.Content)); err != nil {
		return tools.Fail(env, "write", err.Error()), nil
	}
	return tools.OK(env, "write", "wrote "+a.Path, 0, 0), nil
}
