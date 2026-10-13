package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

const maxReadLines = 400

type readTool struct{}

func NewRead() tools.Tool { return &readTool{} }

func (readTool) Name() string { return "read" }

func (readTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "read",
		Description: "Read a line range from a file",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"start_line":{"type":"integer"},"end_line":{"type":"integer"}},"required":["path","start_line","end_line"]}`),
	}
}

type readArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func (readTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	_ = env.EnsureSecurity()
	var a readArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "read", "invalid read args"), nil
	}
	if env.Policy.DenyRead(a.Path) {
		return tools.Fail(env, "read", security.DenyReadMessage), nil
	}
	if a.EndLine < a.StartLine {
		return tools.Fail(env, "read", "end_line before start_line"), nil
	}
	if a.EndLine-a.StartLine+1 > maxReadLines {
		return tools.Fail(env, "read", fmt.Sprintf("max %d lines per read", maxReadLines)), nil
	}
	b, err := env.Workspace.ReadFile(a.Path)
	if err != nil {
		return tools.Fail(env, "read", err.Error()), nil
	}
	env.Workspace.NoteReadRange(a.Path, b, a.StartLine, a.EndLine)
	lines := strings.Split(string(b), "\n")
	start := a.StartLine - 1
	if start < 0 {
		start = 0
	}
	end := a.EndLine
	if end > len(lines) {
		end = len(lines)
	}
	if start >= len(lines) {
		return tools.OK(env, "read", "", 0, maxReadLines), nil
	}
	out := strings.Join(lines[start:end], "\n")
	return tools.OK(env, "read", out, 0, maxReadLines), nil
}
