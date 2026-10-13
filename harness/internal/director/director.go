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
	"github.com/PLACEHOLDER/thee/harness/internal/provider/openai"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/toolcall"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
	"github.com/PLACEHOLDER/thee/harness/internal/tools/impl"
	"github.com/PLACEHOLDER/thee/harness/internal/verifier"
	"github.com/PLACEHOLDER/thee/harness/internal/workspace"
)

const sessionID = "test-session"

// Outcome is the terminal session result.
type Outcome struct {
	State    string
	Reason   string
	ExitCode int
}

// Config drives a director run.
type Config struct {
	Goal            string
	Workspace       string
	Trusted         bool
	TheeVersion     string
	Provider        provider.Provider
	Manifest        *config.Manifest
	MaxRounds       int
	TracePath       string
	Caps            SessionCaps
	VerifyCmd       string
	RegisterAllTools bool
}

// Run executes the session state machine.
func Run(ctx context.Context, cfg Config) (Outcome, error) {
	trusted := security.WorkspaceTrusted(cfg.Workspace, cfg.Trusted)
	if !trusted {
		return Outcome{State: StateFailed, Reason: "workspace_untrusted", ExitCode: 3}, nil
	}
	ws, err := workspace.Open(cfg.Workspace)
	if err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	model, err := cfg.Manifest.ResolveRole("worker")
	if err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	eff, _ := config.LoadEffective(ws.Root())
	traceDir := filepath.Join(ws.Root(), ".thee", "sessions", sessionID)
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	traceFile := cfg.TracePath
	if traceFile == "" {
		traceFile = filepath.Join(traceDir, "trace.jsonl")
	}
	tw, err := observability.NewWriter(traceFile)
	if err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	defer tw.Close()

	_ = tw.Append(observability.SessionStartEvent(cfg.TheeVersion, ws.Root()))
	_ = workspace.EnsureSessionBase(ws.Root(), sessionID)

	reg := tools.NewRegistry()
	reg.Register(impl.NewRead())
	reg.Register(impl.NewPatch())
	reg.Register(impl.NewFinish())
	if cfg.RegisterAllTools {
		reg.Register(impl.NewSearch())
		reg.Register(impl.NewList())
		reg.Register(impl.NewWrite())
		reg.Register(impl.NewDelete())
		reg.Register(impl.NewRepoMap())
		reg.Register(impl.NewTodo())
		reg.Register(impl.NewAskUser())
		cp := NewCheckpointStore(ws.Root(), sessionID)
		reg.Register(impl.NewCheckpoint(cp.RestoreLatest))
	}

	session := &tools.SessionState{}
	toolEnv := &tools.Env{Workspace: ws, Trusted: trusted, Session: session}
	estimator := contextengine.NewEstimator()
	pol := security.NewPolicy()
	ign, _ := security.LoadIgnore(ws.Root(), pol)
	cpStore := NewCheckpointStore(ws.Root(), sessionID)

	caps := cfg.Caps
	if caps.MaxToolRounds == 0 {
		caps = DefaultCaps()
	}
	if cfg.MaxRounds > 0 {
		caps.MaxToolRounds = cfg.MaxRounds
	}
	budget := NewBudget(caps)
	repair := RepairState{}
	loops := &loopDetector{}
	phase := tools.PhaseEditing

	verifyOpt := verifier.ResolveOptions{
		CLIOverride: cfg.VerifyCmd,
		ConfigCmd:   eff.VerifyCommand,
	}
	if verifyOpt.CLIOverride == "" {
		verifyOpt.CLIOverride = config.VerifyCommandFromEnv()
	}
	verifyArgv, verifyOK := verifier.Resolve(ws.Root(), verifyOpt)
	if !verifyOK {
		return end(tw, Outcome{State: StateFailed, Reason: "no_verify_command", ExitCode: 3}), nil
	}

	var baseline verifier.BaselineCache
	baseline.StartBaseline(ctx, ws.Root(), verifyArgv, eff.Baseline)

	state := StateActing
	history := []provider.Message{}
	firstTurn := true
	baselineNote := ""

	for round := 0; round < caps.MaxToolRounds && state != StateDone && state != StateFailed && state != StateCancelled; round++ {
		switch state {
		case StateActing:
			if firstTurn {
				baselineNote = baseline.Note()
				firstTurn = false
			}
			turn := contextengine.BuildTurn(contextengine.TurnInput{
				Goal:          cfg.Goal,
				History:       history,
				ToolProtocol:  model.Capabilities.ToolProtocol,
				PromptVariant: model.Capabilities.PromptVariant,
				ToolSpecs:     reg.Specs(),
				WorkspaceRoot: ws.Root(),
				Trusted:       trusted,
				Phase:         phase,
				ContextWindow: model.Capabilities.ContextWindow,
				Estimator:     estimator,
				Policy:        pol,
				Ignore:        ign,
			})
			if baselineNote != "" {
				turn.Messages[len(turn.Messages)-1].Content = baselineNote + turn.Messages[len(turn.Messages)-1].Content
				baselineNote = ""
			}
			if !budget.FitsModelCall(turn.EstimatedInputTokens) {
				if !budget.Compacted {
					history = contextengine.CompactHistory(history, turn.EstimatedInputTokens/2, estimator)
					budget.Compacted = true
					continue
				}
				return end(tw, Outcome{State: StateFailed, Reason: "budget_exhausted", ExitCode: 1}), nil
			}
			req := provider.Request{
				ModelID:         model.Model,
				Messages:        turn.Messages,
				Tools:           turn.Tools,
				MaxOutputTokens: caps.MaxOutputTokens,
			}
			out, resp, err := toolcall.Next(ctx, cfg.Provider, req, toolcall.DispatchInput{
				ToolProtocol: model.Capabilities.ToolProtocol,
				Phase:        phase,
				Specs:        reg.Specs(),
			})
			if err != nil {
				if repair.ConsecutiveFailure(err) {
					if session.EscalationsUsed < caps.MaxEscalations {
						transition(tw, state, StateEscalating, "tool_parse")
						session.EscalationsUsed++
						transition(tw, StateEscalating, StateActing, "same_model")
						repair = RepairState{}
						continue
					}
					return end(tw, Outcome{State: StateFailed, Reason: "tool_parse_exhausted", ExitCode: 1}), nil
				}
				if repair.ShouldRepair(err) {
					history = append(history, repair.RepairMessage())
					continue
				}
				return end(tw, Outcome{State: StateFailed, Reason: "no_tool_call", ExitCode: 1}), nil
			}
			repair = RepairState{}
			estimator.ObserveUsage(resp.Usage.InputTokens, turn.EstimatedInputTokens)
			budget.Charge(resp.Usage.InputTokens, resp.Usage.OutputTokens)
			if openai.TruncatedByServer(turn.EstimatedInputTokens, resp.Usage.InputTokens) {
				_ = tw.Append(observability.TruncatedByServerEvent())
			}
			if nudge, fire := loops.CheckRepeated(ws.Changed()); fire {
				loopEvent(tw, "repeated_call")
				if loops.ShouldEscalate() {
					if session.EscalationsUsed < caps.MaxEscalations {
						transition(tw, state, StateEscalating, "loop")
						session.EscalationsUsed++
						transition(tw, StateEscalating, StateActing, "same_model")
					} else {
						return end(tw, Outcome{State: StateFailed, Reason: "loop_detected", ExitCode: 1}), nil
					}
				} else {
					history = append(history, provider.Message{Role: "user", Content: nudge})
				}
			}
			history = contextengine.AppendAssistantToolCalls(history, out.Calls)
			transition(tw, state, StateTooling, "")
			state = StateTooling
			session.ToolRounds++

			for _, call := range out.Calls {
				if out.Salvaged {
					_ = tw.Append(observability.ToolCallSalvagedEvent(call.Name))
				} else {
					_ = tw.Append(observability.ToolCallEvent(call.Name))
				}
				loops.RecordToolCall(call.Name, call.Arguments)
				if err := reg.Check(call.Name, phase); err != nil {
					return end(tw, Outcome{State: StateFailed, Reason: "phase_violation", ExitCode: 1}), nil
				}
				tool, err := reg.Get(call.Name)
				if err != nil {
					return end(tw, Outcome{State: StateFailed, Reason: "unknown_tool", ExitCode: 1}), nil
				}
				res, err := tool.Run(ctx, toolEnv, call.Arguments)
				if err != nil {
					return end(tw, Outcome{State: StateFailed, Reason: "tool_error", ExitCode: 1}), err
				}
				_ = tw.Append(observability.ToolResultEvent(res.OK))
				history = contextengine.AppendToolResult(history, call.ID, call.Name, res.Text)
				if call.Name == "patch" && res.OK {
					var pa struct {
						Path string `json:"path"`
					}
					_ = json.Unmarshal(call.Arguments, &pa)
					if pa.Path != "" {
						_ = cpStore.Save(pa.Path)
					}
				}
				if call.Name == "finish" {
					if session.PendingVerify {
						transition(tw, state, StateVerifying, "finish")
						state = StateVerifying
					} else {
						transition(tw, state, StateDone, "finish")
						state = StateDone
					}
				} else {
					transition(tw, state, StateActing, "")
					state = StateActing
				}
			}

		case StateVerifying:
			session.VerifyAttempts++
			if session.VerifyAttempts > caps.MaxVerifyAttempts {
				return end(tw, Outcome{State: StateFailed, Reason: "verify_exhausted", ExitCode: 1}), nil
			}
			vres, err := verifier.Run(ctx, ws.Root(), verifyArgv)
			if err != nil {
				return end(tw, Outcome{State: StateFailed, Reason: "verify_error", ExitCode: 1}), err
			}
			_ = tw.Append(observability.VerifyRunEvent(vres.Command, vres.ExitCode, vres.Passed, vres.DurationMS))
			if vres.Passed {
				session.PendingVerify = false
				transition(tw, state, StateDone, "verified")
				state = StateDone
			} else {
				bundle := verifier.ParseGoTest(vres.Excerpt)
				history = appendVerifyFailure(history, verifier.FormatVerifyFailure(vres, bundle))
				transition(tw, state, StateActing, "verify_fail")
				state = StateActing
			}

		default:
			return end(tw, Outcome{State: StateFailed, Reason: "bad_state", ExitCode: 1}), nil
		}
	}
	if state == StateDone {
		return end(tw, Outcome{State: StateDone, Reason: "verified", ExitCode: 0}), nil
	}
	return end(tw, Outcome{State: StateFailed, Reason: "tool_rounds_exhausted", ExitCode: 1}), nil
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
