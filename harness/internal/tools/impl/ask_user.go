package impl

import (
	"context"
	"encoding/json"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

const maxAskPerGoal = 3

type askUserTool struct{}

func NewAskUser() tools.Tool { return &askUserTool{} }

func (askUserTool) Name() string { return "ask_user" }

func (askUserTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "ask_user",
		Description: "Ask the user a clarifying question",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"}},"required":["question"]}`),
	}
}

type askUserArgs struct {
	Question string `json:"question"`
}

func (askUserTool) Run(ctx context.Context, env *tools.Env, args json.RawMessage) (tools.Result, error) {
	var a askUserArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return tools.Fail(env, "ask_user", "invalid args"), nil
	}
	if env.Session == nil {
		return tools.Fail(env, "ask_user", "no session"), nil
	}
	env.Session.AskUserUsed++
	if env.Session.AskUserUsed > maxAskPerGoal {
		return tools.OK(env, "ask_user", "decide and state your assumption in finish summary", 0, 0), nil
	}
	msg := "No user is present: choose the safest assumption and state it in the finish summary."
	return tools.OK(env, "ask_user", msg, 0, 0), nil
}
