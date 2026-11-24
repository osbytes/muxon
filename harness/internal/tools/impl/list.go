package impl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

const maxListEntries = 200

type listTool struct{}

// NewList returns the list tool.
func NewList() tools.Tool { return &listTool{} }

func (listTool) Name() string { return "list" }

func (listTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "list",
		Description: "List directory entries",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"glob":{"type":"string"}},"required":["path"]}`),
	}
}

type listArgs struct {
	Path string `json:"path"`
	Glob string `json:"glob"`
}

func (listTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	_ = env.EnsureSecurity()
	var a listArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "list", "invalid list args"), nil
	}
	abs, err := env.Workspace.Resolve(a.Path)
	if err != nil {
		return tools.Fail(env, "list", err.Error()), nil
	}
	st, err := os.Stat(abs)
	if err != nil {
		return tools.Fail(env, "list", err.Error()), nil
	}
	if !st.IsDir() {
		return tools.Fail(env, "list", "not a directory"), nil
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return tools.Fail(env, "list", err.Error()), nil
	}
	var rows []security.ListEntry
	for _, e := range entries {
		name := e.Name()
		rel := filepath.ToSlash(filepath.Join(a.Path, name))
		if env.Ignore != nil && env.Ignore.SkipInSearchList(rel) && !security.IsBulkDir(name) {
			continue
		}
		if security.IsBulkDir(name) {
			count := security.CountDirEntries(filepath.Join(abs, name))
			rows = append(rows, security.ListEntry{Name: rel + "/", Count: count})
			continue
		}
		rows = append(rows, security.ListEntry{Name: rel})
		if len(rows) >= maxListEntries {
			break
		}
	}
	rows = security.CollapseBulkEntries(rows)
	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	body := strings.Join(names, "\n")
	return tools.OK(env, "list", body, 0, maxListEntries), nil
}
