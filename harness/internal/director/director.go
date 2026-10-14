package director

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/contextengine"
	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/openai"
	"github.com/PLACEHOLDER/thee/harness/internal/router"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/toolcall"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
	"github.com/PLACEHOLDER/thee/harness/internal/tools/impl"
	"github.com/PLACEHOLDER/thee/harness/internal/verifier"
	"github.com/PLACEHOLDER/thee/harness/internal/workspace"
)

const defaultSessionID = "test-session"

// Outcome is the terminal session result.
type Outcome struct {
	State    string
	Reason   string
	ExitCode int
}

// Config drives a director run.
type Config struct {
	Goal                string
	Workspace           string
	Trusted             bool
	TheeVersion         string
	Provider            provider.Provider
	Manifest            *config.Manifest
	MaxRounds           int
	TracePath           string
	SessionID           string
	Caps                SessionCaps
	VerifyCmd           string
	RegisterAllTools    bool
	WorkerModelOverride string
	NoPlan              bool
	OnEvent             func(observability.Event)
	ForceEscalate       bool
	Interactive         bool
}

// Run executes the session state machine.
func Run(ctx context.Context, cfg Config) (Outcome, error) {
	if err := ctx.Err(); err != nil {
		return Outcome{State: StateCancelled, Reason: "user_cancel", ExitCode: 130}, nil
	}
	trusted := security.WorkspaceTrusted(cfg.Workspace, cfg.Trusted)
	if !trusted {
		return Outcome{State: StateFailed, Reason: "workspace_untrusted", ExitCode: 3}, nil
	}
	ws, err := workspace.Open(cfg.Workspace)
	if err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	manifest := cfg.Manifest
	if manifest == nil {
		manifest, err = config.LoadMergedManifest()
		if err != nil {
			return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
		}
	}
	workerBind, err := router.ResolveWorker(manifest, cfg.WorkerModelOverride)
	if err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	model := workerBind.Model
	activeProv := cfg.Provider
	if activeProv == nil {
		activeProv, err = router.NewProvider(workerBind)
		if err != nil {
			return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
		}
	}
	var escalatedProv provider.Provider
	var escalatedModel *config.ModelDef
	hasEscalatedRole := false
	if eb, err := router.ResolveEscalated(manifest); err == nil {
		escalatedModel = eb.Model
		if cfg.Provider != nil {
			escalatedProv = cfg.Provider
			hasEscalatedRole = true
		} else if p, err := router.NewProvider(eb); err == nil {
			escalatedProv = p
			hasEscalatedRole = true
		}
	}
	var routerProv provider.Provider
	if rb, err := router.ResolveRole(manifest, "router"); err == nil {
		routerProv, _ = router.NewProvider(rb)
	}

	eff, _ := config.LoadEffective(ws.Root())
	uc, _ := config.LoadUserConfig()
	sid := cfg.SessionID
	if sid == "" {
		sid = defaultSessionID
	}
	traceDir := filepath.Join(ws.Root(), ".thee", "sessions", sid)
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

	session := &tools.SessionState{}
	var totals observability.UsageTotals
	var malformed int
	var loopsDetected int

	emit := func(ev observability.Event) {
		_ = tw.Append(ev)
		if cfg.OnEvent != nil {
			cfg.OnEvent(ev)
		}
	}
	finish := func(o Outcome) Outcome {
		emit(observability.SessionEndEvent(o.State, o.Reason, totals, session.ToolRounds, session.VerifyAttempts, session.EscalationsUsed, malformed, loopsDetected))
		return o
	}

	emit(observability.SessionStartEvent(cfg.TheeVersion, ws.Root()))
	_ = workspace.EnsureSessionBase(ws.Root(), sid)

	class := router.ClassifyGoal(ctx, cfg.Goal, manifest, routerProv)
	emit(observability.GoalStartEvent(class.TaskType, class.Confidence))

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
		reg.Register(impl.NewGit())
		if eff.FilterCommand != "" {
			reg.Register(impl.NewTest())
		}
		cp := NewCheckpointStore(ws.Root(), sid)
		reg.Register(impl.NewCheckpoint(cp.RestoreLatest))
	}

	toolEnv := &tools.Env{
		Workspace:      ws,
		Trusted:        trusted,
		Session:        session,
		Headless:       !cfg.Interactive,
		AllowGitCommit: eff.AllowCommit,
		FilterCommand:  eff.FilterCommand,
	}
	estimator := contextengine.NewEstimator()
	pol := security.NewPolicy()
	ign, _ := security.LoadIgnore(ws.Root(), pol)
	cpStore := NewCheckpointStore(ws.Root(), sid)

	caps := cfg.Caps
	if caps.MaxToolRounds == 0 {
		caps = DefaultCaps()
	}
	if uc.Session.MaxToolRounds > 0 && cfg.MaxRounds == 0 {
		caps.MaxToolRounds = uc.Session.MaxToolRounds
	}
	if uc.Session.CostBudgetUSD > 0 && caps.CostBudgetUSD == 0 {
		caps.CostBudgetUSD = uc.Session.CostBudgetUSD
	}
	if uc.Session.MaxEscalations > 0 && caps.MaxEscalations == 0 {
		caps.MaxEscalations = uc.Session.MaxEscalations
	}
	if uc.Session.EscalateAfterVerifyFails > 0 && caps.EscalateAfterVerifyFails == 0 {
		caps.EscalateAfterVerifyFails = uc.Session.EscalateAfterVerifyFails
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
		return finish(Outcome{State: StateFailed, Reason: "no_verify_command", ExitCode: 3}), nil
	}

	var baseline verifier.BaselineCache
	baseline.StartBaseline(ctx, ws.Root(), verifyArgv, eff.Baseline)

	state := StateActing
	history := []provider.Message{}
	firstTurn := true
	baselineNote := ""

	tryEscalate := func(in router.EscalationInput) bool {
		in.EscalationGeneration = session.EscalationsUsed
		in.MaxEscalations = caps.MaxEscalations
		in.EscalateAfterVerify = caps.EscalateAfterVerifyFails
		in.MaxToolRounds = caps.MaxToolRounds
		in.ToolRounds = session.ToolRounds
		in.VerifyPassed = session.EverVerifyPassed
		in.HasEscalatedRole = hasEscalatedRole && escalatedProv != nil
		in.RemainingCostUSD = budget.RemainingCostUSD()
		dec := router.ShouldEscalate(in)
		if !dec.Escalate {
			return false
		}
		if escalatedProv == nil || escalatedModel == nil {
			return false
		}
		if session.EscalationsUsed >= caps.MaxEscalations {
			return false
		}
		from := model.Model
		emit(observability.EscalateEvent(from, escalatedModel.Model, dec.Reason, session.EscalationsUsed+1))
		emit(observability.StateTransitionEvent(state, StateEscalating, dec.Reason))
		session.EscalationsUsed++
		model = escalatedModel
		activeProv = escalatedProv
		emit(observability.StateTransitionEvent(StateEscalating, StateActing, "model_swapped"))
		return true
	}

	for round := 0; round < caps.MaxToolRounds && state != StateDone && state != StateFailed && state != StateCancelled; round++ {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				return finish(Outcome{State: StateCancelled, Reason: "user_cancel", ExitCode: 130}), nil
			}
			return finish(Outcome{State: StateFailed, Reason: "setup", ExitCode: 1}), err
		}
		switch state {
		case StateActing:
			if cfg.ForceEscalate {
				tryEscalate(router.EscalationInput{ForceUser: true})
				cfg.ForceEscalate = false
			}
			if class.LowConfidence() {
				tryEscalate(router.EscalationInput{LowRouterConfidence: true})
			}
			if session.ToolRounds >= (caps.MaxToolRounds*60)/100 && !session.EverVerifyPassed {
				tryEscalate(router.EscalationInput{})
			}
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
				return finish(Outcome{State: StateFailed, Reason: "budget_exhausted", ExitCode: 1}), nil
			}
			rates := router.RatesFromModel(model)
			estUSD := (float64(turn.EstimatedInputTokens)*rates.InputPerM + float64(caps.MaxOutputTokens)*rates.OutputPerM) / 1e6
			if !budget.FitsCost(estUSD) {
				return finish(Outcome{State: StateFailed, Reason: "budget_exhausted", ExitCode: 1}), nil
			}
			req := provider.Request{
				ModelID:         model.Model,
				Messages:        turn.Messages,
				Tools:           turn.Tools,
				MaxOutputTokens: caps.MaxOutputTokens,
			}
			out, resp, err := toolcall.Next(ctx, activeProv, req, toolcall.DispatchInput{
				ToolProtocol: model.Capabilities.ToolProtocol,
				Phase:        phase,
				Specs:        reg.Specs(),
			})
			if err != nil {
				malformed++
				if repair.ConsecutiveFailure(err) {
					if tryEscalate(router.EscalationInput{ConsecutiveMalformed: true}) {
						repair = RepairState{}
						continue
					}
					return finish(Outcome{State: StateFailed, Reason: "tool_parse_exhausted", ExitCode: 1}), nil
				}
				if repair.ShouldRepair(err) {
					history = append(history, repair.RepairMessage())
					continue
				}
				return finish(Outcome{State: StateFailed, Reason: "no_tool_call", ExitCode: 1}), nil
			}
			repair = RepairState{}
			estimator.ObserveUsage(resp.Usage.InputTokens, turn.EstimatedInputTokens)
			budget.Charge(resp.Usage.InputTokens, resp.Usage.OutputTokens)
			callCost, _ := router.CostUSD(rates, resp.Usage)
			budget.ChargeCost(callCost)
			totals.Add(resp.Usage, callCost)
			emit(observability.ModelResponseEvent(callCost, resp.Usage))
			if openai.TruncatedByServer(turn.EstimatedInputTokens, resp.Usage.InputTokens) {
				emit(observability.TruncatedByServerEvent())
			}
			if nudge, fire := loops.CheckRepeated(ws.Changed()); fire {
				loopsDetected++
				emit(observability.Event{Type: "loop.detected", Reason: "repeated_call"})
				if loops.ShouldEscalate() {
					if tryEscalate(router.EscalationInput{LoopSecondFire: true}) {
						continue
					}
					return finish(Outcome{State: StateFailed, Reason: "loop_detected", ExitCode: 1}), nil
				}
				history = append(history, provider.Message{Role: "user", Content: nudge})
			}
			history = contextengine.AppendAssistantToolCalls(history, out.Calls)
			emit(observability.StateTransitionEvent(state, StateTooling, ""))
			state = StateTooling
			session.ToolRounds++

			for _, call := range out.Calls {
				if out.Salvaged {
					emit(observability.ToolCallSalvagedEvent(call.Name))
				} else {
					emit(observability.ToolCallEvent(call.Name))
				}
				loops.RecordToolCall(call.Name, call.Arguments)
				if err := reg.Check(call.Name, phase); err != nil {
					return finish(Outcome{State: StateFailed, Reason: "phase_violation", ExitCode: 1}), nil
				}
				tool, err := reg.Get(call.Name)
				if err != nil {
					return finish(Outcome{State: StateFailed, Reason: "unknown_tool", ExitCode: 1}), nil
				}
				res, err := tool.Run(ctx, toolEnv, call.Arguments)
				if err != nil {
					return finish(Outcome{State: StateFailed, Reason: "tool_error", ExitCode: 1}), err
				}
				emit(observability.ToolResultEvent(res.OK))
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
						emit(observability.StateTransitionEvent(state, StateVerifying, "finish"))
						state = StateVerifying
					} else {
						emit(observability.StateTransitionEvent(state, StateDone, "finish"))
						state = StateDone
					}
				} else {
					emit(observability.StateTransitionEvent(state, StateActing, ""))
					state = StateActing
				}
			}

		case StateVerifying:
			session.VerifyAttempts++
			if session.VerifyAttempts > caps.MaxVerifyAttempts {
				return finish(Outcome{State: StateFailed, Reason: "verify_exhausted", ExitCode: 1}), nil
			}
			vres, err := verifier.Run(ctx, ws.Root(), verifyArgv)
			if err != nil {
				return finish(Outcome{State: StateFailed, Reason: "verify_error", ExitCode: 1}), err
			}
			emit(observability.VerifyRunEvent(vres.Command, vres.ExitCode, vres.Passed, vres.DurationMS))
			if vres.Passed {
				session.PendingVerify = false
				session.ConsecutiveVerifyFails = 0
				session.EverVerifyPassed = true
				emit(observability.StateTransitionEvent(state, StateDone, "verified"))
				state = StateDone
			} else {
				session.ConsecutiveVerifyFails++
				if tryEscalate(router.EscalationInput{ConsecutiveVerifyFails: session.ConsecutiveVerifyFails}) {
					bundle := verifier.ParseGoTest(vres.Excerpt)
					history = appendVerifyFailure(history, verifier.FormatVerifyFailure(vres, bundle))
					state = StateActing
					continue
				}
				bundle := verifier.ParseGoTest(vres.Excerpt)
				history = appendVerifyFailure(history, verifier.FormatVerifyFailure(vres, bundle))
				emit(observability.StateTransitionEvent(state, StateActing, "verify_fail"))
				state = StateActing
			}

		default:
			return finish(Outcome{State: StateFailed, Reason: "bad_state", ExitCode: 1}), nil
		}
	}
	if state == StateDone {
		return finish(Outcome{State: StateDone, Reason: "verified", ExitCode: 0}), nil
	}
	return finish(Outcome{State: StateFailed, Reason: "tool_rounds_exhausted", ExitCode: 1}), nil
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
