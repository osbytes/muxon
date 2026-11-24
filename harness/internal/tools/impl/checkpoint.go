package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

type checkpointTool struct {
	store interface {
		RestoreLatest(string) error
	}
}

// NewCheckpoint creates the checkpoint tool with an optional restore backend.
func NewCheckpoint(restore func(string) error) tools.Tool {
	return &checkpointTool{store: restoreBackend{restore}}
}

type restoreBackend struct {
	fn func(string) error
}

func (r restoreBackend) RestoreLatest(path string) error {
	if r.fn == nil {
		return nil
	}
	return r.fn(path)
}

func (checkpointTool) Name() string { return "checkpoint" }

func (checkpointTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "checkpoint",
		Description: "List or restore workspace checkpoints",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"},"path":{"type":"string"}},"required":["action"]}`),
	}
}

type checkpointArgs struct {
	Action string `json:"action"`
	Path   string `json:"path"`
}

func (c checkpointTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	var a checkpointArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "checkpoint", "invalid args"), nil
	}
	switch a.Action {
	case "list":
		return tools.OK(env, "checkpoint", "checkpoints stored per session", 0, 0), nil
	case "restore":
		if c.store == nil || a.Path == "" {
			return tools.Fail(env, "checkpoint", "restore unavailable"), nil
		}
		if err := c.store.RestoreLatest(a.Path); err != nil {
			return tools.Fail(env, "checkpoint", err.Error()), nil
		}
		return tools.OK(env, "checkpoint", "restored "+a.Path, 0, 0), nil
	default:
		return tools.Fail(env, "checkpoint", "unknown action"), nil
	}
}
