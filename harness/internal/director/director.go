package director

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/contextengine"
	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/toolcall"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
	"github.com/PLACEHOLDER/thee/harness/internal/tools/impl"
	"github.com/PLACEHOLDER/thee/harness/internal/verifier"
	"github.com/PLACEHOLDER/thee/harness/internal/workspace"
)

// Outcome is the terminal session result.
type Outcome struct {
	State    string
	Reason   string
	ExitCode int
}

// Config drives a director run.
type Config struct {
	Goal        string
	Workspace   string
	Trusted     bool
	TheeVersion string
	Provider    provider.Provider
	Manifest    *config.Manifest
	MaxRounds   int
	TracePath   string
}

// Run executes the M0 state machine subset.
func Run(ctx context.Context, cfg Config) (Outcome, error) {
	if !cfg.Trusted {
		return Outcome{State: "failed", Reason: "workspace_untrusted", ExitCode: 3}, nil
	}
	ws, err := workspace.Open(cfg.Workspace)
	if err != nil {
		return Outcome{State: "failed", Reason: "setup", ExitCode: 3}, err
	}
	model, err := cfg.Manifest.ResolveRole("worker")
	if err != nil {
		return Outcome{State: "failed", Reason: "setup", ExitCode: 3}, err
	}
	traceDir := filepath.Join(ws.Root(), ".thee", "sessions", "test-session")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		return Outcome{State: "failed", Reason: "setup", ExitCode: 3}, err
	}
	traceFile := cfg.TracePath
	if traceFile == "" {
		traceFile = filepath.Join(traceDir, "trace.jsonl")
	}
	tw, err := observability.NewWriter(traceFile)
	if err != nil {
		return Outcome{State: "failed", Reason: "setup", ExitCode: 3}, err
	}
	defer tw.Close()

	_ = tw.Append(observability.SessionStartEvent(cfg.TheeVersion, ws.Root()))

	reg := tools.NewRegistry()
	reg.Register(impl.NewRead())
	reg.Register(impl.NewPatch())
	reg.Register(impl.NewFinish())

	session := &tools.SessionState{}
	toolEnv := &tools.Env{Workspace: ws, Trusted: cfg.Trusted, Session: session}
	estimator := contextengine.NewEstimator()
	pol := security.NewPolicy()
	ign, _ := security.LoadIgnore(ws.Root(), pol)

	state := "acting"
	history := []provider.Message{}
	phase := tools.PhaseEditing
	maxRounds := cfg.MaxRounds
	if maxRounds == 0 {
		maxRounds = 20
	}

	for round := 0; round < maxRounds && state != "done"; round++ {
		switch state {
		case "acting":
			turn := contextengine.BuildTurn(contextengine.TurnInput{
				Goal:          cfg.Goal,
				History:       history,
				ToolProtocol:  model.Capabilities.ToolProtocol,
				PromptVariant: model.Capabilities.PromptVariant,
				ToolSpecs:     reg.Specs(),
				WorkspaceRoot: ws.Root(),
				Trusted:       cfg.Trusted,
				Phase:         phase,
				ContextWindow: model.Capabilities.ContextWindow,
				Estimator:     estimator,
				Policy:        pol,
				Ignore:        ign,
			})
			req := provider.Request{
				ModelID:  model.Model,
				Messages: turn.Messages,
				Tools:    turn.Tools,
			}
			out, _, err := toolcall.Next(ctx, cfg.Provider, req)
			if err != nil {
				return end(tw, Outcome{State: "failed", Reason: "no_tool_call", ExitCode: 1}), nil
			}
			history = contextengine.AppendAssistantToolCalls(history, out.Calls)
			state = "tooling"
			_ = tw.Append(observability.StateTransitionEvent("acting", "tooling", ""))

			for _, call := range out.Calls {
				_ = tw.Append(observability.ToolCallEvent(call.Name))
				tool, err := reg.Get(call.Name)
				if err != nil {
					return end(tw, Outcome{State: "failed", Reason: "unknown_tool", ExitCode: 1}), nil
				}
				res, err := tool.Run(ctx, toolEnv, call.Arguments)
				if err != nil {
					return end(tw, Outcome{State: "failed", Reason: "tool_error", ExitCode: 1}), err
				}
				_ = tw.Append(observability.ToolResultEvent(res.OK))
				history = contextengine.AppendToolResult(history, call.ID, call.Name, res.Text)
				if call.Name == "finish" {
					if session.PendingVerify {
						state = "verifying"
						_ = tw.Append(observability.StateTransitionEvent("tooling", "verifying", "finish"))
					} else {
						state = "done"
						_ = tw.Append(observability.StateTransitionEvent("tooling", "done", "finish"))
					}
				} else {
					state = "acting"
					_ = tw.Append(observability.StateTransitionEvent("tooling", "acting", ""))
				}
			}

		case "verifying":
			cmd, ok := verifier.ResolveCommand(ws.Root())
			if !ok {
				return end(tw, Outcome{State: "failed", Reason: "no_verify_command", ExitCode: 3}), nil
			}
			vres, err := verifier.Run(ctx, ws.Root(), cmd)
			if err != nil {
				return end(tw, Outcome{State: "failed", Reason: "verify_error", ExitCode: 1}), err
			}
			_ = tw.Append(observability.VerifyRunEvent(vres.Command, vres.ExitCode, vres.Passed, vres.DurationMS))
			if vres.Passed {
				session.PendingVerify = false
				state = "done"
				_ = tw.Append(observability.StateTransitionEvent("verifying", "done", "verified"))
			} else {
				state = "acting"
				history = append(history, provider.Message{
					Role:    "user",
					Content: "verify failed:\n" + vres.Excerpt,
				})
				_ = tw.Append(observability.StateTransitionEvent("verifying", "acting", "verify_fail"))
			}
		default:
			return end(tw, Outcome{State: "failed", Reason: "bad_state", ExitCode: 1}), nil
		}
	}
	if state == "done" {
		return end(tw, Outcome{State: "done", Reason: "verified", ExitCode: 0}), nil
	}
	return end(tw, Outcome{State: "failed", Reason: "tool_rounds_exhausted", ExitCode: 1}), nil
}

func end(tw *observability.Writer, o Outcome) Outcome {
	_ = tw.Append(observability.SessionEndEvent(o.State, o.Reason))
	return o
}

// M0ScriptedResponses builds the default fixture-fix script.
func M0ScriptedResponses() []provider.Response {
	readArgs, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "start_line": 1, "end_line": 10,
	})
	patchArgs, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{
			{"search": "return a - b", "replace": "return a + b"},
		},
	})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "fixed sum"})
	return []provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "read", Arguments: readArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "2", Name: "patch", Arguments: patchArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "3", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	}
}
