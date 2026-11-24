package impl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

const maxOpenTodos = 30

type todoTool struct{}

func NewTodo() tools.Tool { return &todoTool{} }

func (todoTool) Name() string { return "todo" }

func (todoTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "todo",
		Description: "Track open tasks for the goal",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"},"text":{"type":"string"},"id":{"type":"integer"}},"required":["action"]}`),
	}
}

type todoArgs struct {
	Action string `json:"action"`
	Text   string `json:"text"`
	ID     int    `json:"id"`
}

func (todoTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	var a todoArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "todo", "invalid args"), nil
	}
	if env.Session == nil {
		return tools.Fail(env, "todo", "no session"), nil
	}
	switch a.Action {
	case "add":
		if len(env.Session.OpenTodos) >= maxOpenTodos {
			return tools.Fail(env, "todo", "max open todos"), nil
		}
		env.Session.OpenTodos = append(env.Session.OpenTodos, a.Text)
		return tools.OK(env, "todo", "added", 0, 0), nil
	case "done":
		if a.ID <= 0 || a.ID > len(env.Session.OpenTodos) {
			return tools.Fail(env, "todo", "invalid id"), nil
		}
		env.Session.OpenTodos = append(env.Session.OpenTodos[:a.ID-1], env.Session.OpenTodos[a.ID:]...)
		return tools.OK(env, "todo", "done", 0, 0), nil
	case "list":
		body := fmt.Sprintf("%d open", len(env.Session.OpenTodos))
		for i, t := range env.Session.OpenTodos {
			body += fmt.Sprintf("\n%d. %s", i+1, t)
		}
		return tools.OK(env, "todo", body, 0, 0), nil
	default:
		return tools.Fail(env, "todo", "unknown action"), nil
	}
}
