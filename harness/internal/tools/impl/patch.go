package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools/patch"
	"github.com/PLACEHOLDER/muxon/harness/internal/verifier"
)

type patchTool struct{}

func NewPatch() tools.Tool { return &patchTool{} }

func (patchTool) Name() string { return "patch" }

func (patchTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "patch",
		Description: "Patch a file using search_replace or udiff",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"format":{"type":"string"},"diff":{"type":"string"},"replacements":{"type":"array"}},"required":["path"]}`),
	}
}

type patchArgs struct {
	Path         string              `json:"path"`
	Format       string              `json:"format"`
	Diff         string              `json:"diff"`
	Replacements []patch.Replacement `json:"replacements"`
}

func (patchTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	var a patchArgs
	_ = env.EnsureSecurity()
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "patch", "invalid patch args"), nil
	}
	if env.Policy.DenyRead(a.Path) {
		return tools.Fail(env, "patch", security.DenyReadMessage), nil
	}
	if err := env.Workspace.CheckFresh(a.Path); err != nil {
		return tools.Fail(env, "patch", err.Error()), nil
	}
	cur, err := env.Workspace.ReadFile(a.Path)
	if err != nil {
		return tools.Fail(env, "patch", err.Error()), nil
	}
	var newContent []byte
	var strategy string
	switch a.Format {
	case "", "search_replace":
		newContent, strategy, err = patch.ApplySearchReplace(string(cur), a.Replacements)
	case "udiff":
		newContent, strategy, err = patch.ApplyUDiff(string(cur), a.Diff)
	default:
		return tools.Fail(env, "patch", "unknown patch format"), nil
	}
	if err != nil {
		return tools.Fail(env, "patch", err.Error()), nil
	}
	if err := env.Workspace.WriteFile(a.Path, newContent); err != nil {
		return tools.Fail(env, "patch", err.Error()), nil
	}
	verifier.NoteTestEdit(env.Session, a.Path)
	res := tools.OK(env, "patch", "patched "+a.Path+" ("+strategy+")", 0, 0)
	res.ApplyStrategy = strategy
	return res, nil
}
