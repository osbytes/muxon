package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/subproc"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

const maxSearchHits = 50
const maxPreviewChars = 200

type searchTool struct{}

func NewSearch() tools.Tool { return &searchTool{} }

func (searchTool) Name() string { return "search" }

func (searchTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "search",
		Description: "Search workspace with ripgrep",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"},"glob":{"type":"string"},"max_hits":{"type":"integer"}},"required":["pattern"]}`),
	}
}

type searchArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Glob    string `json:"glob"`
	MaxHits int    `json:"max_hits"`
}

func (searchTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	_ = env.EnsureSecurity()
	var a searchArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "search", "invalid search args"), nil
	}
	if _, err := exec.LookPath("rg"); err != nil {
		return tools.Fail(env, "search", "rg not found on PATH"), nil
	}
	limit := a.MaxHits
	if limit <= 0 || limit > maxSearchHits {
		limit = maxSearchHits
	}
	searchPath := a.Path
	if searchPath == "" {
		searchPath = "."
	}
	argv := []string{"rg", "--no-heading", "--line-number", "--max-count", fmt.Sprintf("%d", limit), a.Pattern}
	if a.Glob != "" {
		argv = append(argv, "--glob", a.Glob)
	}
	argv = append(argv, searchPath)
	out, code, err := subproc.Run(ctx, env.Workspace.Root(), argv, nil)
	if err != nil {
		return tools.Fail(env, "search", err.Error()), nil
	}
	if code > 1 {
		return tools.Fail(env, "search", strings.TrimSpace(string(out))), nil
	}
	lines := filterSearchLines(env, string(out))
	body := formatSearchHits(lines)
	return tools.OK(env, "search", body, 0, 0), nil
}

func filterSearchLines(env *tools.Env, raw string) []string {
	var kept []string
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		path := searchLinePath(line)
		if path == "" {
			continue
		}
		if env.Policy.DenyRead(path) {
			continue
		}
		if env.Ignore != nil && env.Ignore.SkipInSearchList(path) {
			continue
		}
		kept = append(kept, line)
		if len(kept) >= maxSearchHits {
			break
		}
	}
	return kept
}

func searchLinePath(line string) string {
	i := strings.Index(line, ":")
	if i <= 0 {
		return ""
	}
	return line[:i]
}

func formatSearchHits(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		preview := line
		if len(preview) > maxPreviewChars {
			preview = preview[:maxPreviewChars] + "..."
		}
		b.WriteString(preview)
		b.WriteByte('\n')
	}
	return b.String()
}
