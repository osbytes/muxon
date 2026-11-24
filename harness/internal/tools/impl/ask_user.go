package impl

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

const (
	maxAskPerGoal  = 3
	maxAskChoices  = 5
	noUserResponse = "No user is present: choose the safest assumption and state it in the finish summary."
)

type askUserTool struct{}

func NewAskUser() tools.Tool { return &askUserTool{} }

func (askUserTool) Name() string { return "ask_user" }

func (askUserTool) Spec() tools.Spec {
	return tools.Spec{
		Name:        "ask_user",
		Description: "Ask the user one short question when the task cannot continue without a fact only they have",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"},"choices":{"type":"array","items":{"type":"string"},"maxItems":5}},"required":["question"]}`),
	}
}

type askUserArgs struct {
	Question string   `json:"question"`
	Choices  []string `json:"choices"`
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
	if env.Headless || env.AskUser == nil {
		return tools.OK(env, "ask_user", noUserResponse, 0, 0), nil
	}
	if len(a.Choices) > maxAskChoices {
		a.Choices = a.Choices[:maxAskChoices]
	}
	answer, err := env.AskUser(ctx, strings.TrimSpace(a.Question), a.Choices)
	if err != nil {
		return tools.Fail(env, "ask_user", "the user did not answer: "+err.Error()), nil
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return tools.OK(env, "ask_user", "The user gave no answer: choose the safest assumption and state it in the finish summary.", 0, 0), nil
	}
	return tools.OK(env, "ask_user", "User answered: "+answer, 0, 0), nil
}
