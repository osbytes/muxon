package impl

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/subproc"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

type gitTool struct{}

// NewGit returns the git tool.
func NewGit() tools.Tool { return &gitTool{} }

func (gitTool) Name() string { return "git" }

func (gitTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "git",
		Description: "status, diff, log, or commit. Never pushes.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"},"message":{"type":"string"},"paths":{"type":"array","items":{"type":"string"}}},"required":["action"]}`),
	}
}

type gitArgs struct {
	Action  string   `json:"action"`
	Message string   `json:"message"`
	Paths   []string `json:"paths"`
}

func (gitTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	var a gitArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "git", "invalid git args"), nil
	}
	switch a.Action {
	case "status":
		return gitRun(ctx, env, []string{"git", "status", "--porcelain"})
	case "diff":
		return gitRun(ctx, env, []string{"git", "diff"})
	case "log":
		return gitRun(ctx, env, []string{"git", "log", "-n", "5", "--oneline"})
	case "worktree":
		return tools.Fail(env, "git", "not available in this build"), nil
	case "commit":
		if env.Headless && !env.AllowGitCommit {
			return tools.Fail(env, "git", "commit disabled"), nil
		}
		if strings.TrimSpace(a.Message) == "" {
			return tools.Fail(env, "git", "commit requires a message"), nil
		}
		pol := security.NewPolicy()
		for _, p := range a.Paths {
			if pol.DenyRead(p) {
				return tools.Fail(env, "git", "path is on the secret deny list"), nil
			}
		}
		argv := []string{"git", "commit", "-m", a.Message}
		if len(a.Paths) == 0 {
			return tools.Fail(env, "git", "commit requires paths"), nil
		}
		add := append([]string{"git", "add", "--"}, a.Paths...)
		if _, _, err := subproc.Run(ctx, env.Workspace.Root(), add, nil); err != nil {
			return tools.Fail(env, "git", err.Error()), nil
		}
		argv = append(argv, "--")
		argv = append(argv, a.Paths...)
		return gitRun(ctx, env, argv)
	default:
		return tools.Fail(env, "git", "unknown git action"), nil
	}
}

func gitRun(ctx context.Context, env *tools.Env, argv []string) (tools.Result, error) {
	out, code, err := subproc.Run(ctx, env.Workspace.Root(), argv, nil)
	text := strings.TrimSpace(string(out))
	if err != nil && code == 0 {
		return tools.Fail(env, "git", err.Error()), nil
	}
	if code != 0 {
		return tools.Fail(env, "git", text), nil
	}
	return tools.OK(env, "git", text, 0, 0), nil
}
