package impl

import (
	"context"
	"encoding/json"
	"os"

	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type deleteTool struct{}

func NewDelete() tools.Tool { return &deleteTool{} }

func (deleteTool) Name() string { return "delete" }

func (deleteTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "delete",
		Description: "Delete or move a file",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"dest":{"type":"string"}},"required":["path"]}`),
	}
}

type deleteArgs struct {
	Path string `json:"path"`
	Dest string `json:"dest"`
}

func (deleteTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	_ = env.EnsureSecurity()
	var a deleteArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "delete", "invalid delete args"), nil
	}
	if env.Policy.DenyRead(a.Path) {
		return tools.Fail(env, "delete", security.DenyReadMessage), nil
	}
	srcAbs, err := env.Workspace.Resolve(a.Path)
	if err != nil {
		return tools.Fail(env, "delete", err.Error()), nil
	}
	if a.Dest != "" {
		dstAbs, err := env.Workspace.Resolve(a.Dest)
		if err != nil {
			return tools.Fail(env, "delete", err.Error()), nil
		}
		if err := os.Rename(srcAbs, dstAbs); err != nil {
			return tools.Fail(env, "delete", err.Error()), nil
		}
		return tools.OK(env, "delete", "moved "+a.Path+" to "+a.Dest, 0, 0), nil
	}
	st, err := os.Stat(srcAbs)
	if err != nil {
		return tools.Fail(env, "delete", err.Error()), nil
	}
	if st.IsDir() {
		entries, err := os.ReadDir(srcAbs)
		if err != nil {
			return tools.Fail(env, "delete", err.Error()), nil
		}
		if len(entries) > 0 {
			return tools.Fail(env, "delete", "directory not empty"), nil
		}
		if err := os.Remove(srcAbs); err != nil {
			return tools.Fail(env, "delete", err.Error()), nil
		}
		return tools.OK(env, "delete", "removed directory "+a.Path, 0, 0), nil
	}
	if err := os.Remove(srcAbs); err != nil {
		return tools.Fail(env, "delete", err.Error()), nil
	}
	return tools.OK(env, "delete", "removed "+a.Path, 0, 0), nil
}
