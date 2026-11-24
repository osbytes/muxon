package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/thee/harness/internal/index"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type repoMapTool struct{}

// NewRepoMap returns the repomap tool.
func NewRepoMap() tools.Tool { return &repoMapTool{} }

func (repoMapTool) Name() string { return "repo-map" }

func (repoMapTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "repo-map",
		Description: "Ranked symbol map for the repo",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}
}

type repoMapArgs struct {
	Path string `json:"path"`
}

func (repoMapTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	_ = env.EnsureSecurity()
	var a repoMapArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "repo-map", "invalid args"), nil
	}
	root := env.Workspace.Root()
	syms, err := index.LoadOrBuild(root, "", env.Ignore, env.Policy)
	if err != nil {
		return tools.Fail(env, "repo-map", err.Error()), nil
	}
	if a.Path != "" {
		var filtered []index.Symbol
		for _, s := range syms {
			if s.Path == a.Path || hasPrefixPath(s.Path, a.Path) {
				filtered = append(filtered, s)
			}
		}
		syms = filtered
	}
	body := index.FormatSymbols(syms, 16000)
	return tools.OK(env, "repo-map", body, 16000, 0), nil
}

func hasPrefixPath(path, prefix string) bool {
	return len(prefix) > 0 && (path == prefix || len(path) > len(prefix) && path[:len(prefix)+1] == prefix+"/")
}
