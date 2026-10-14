package director

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/contextengine"
	"github.com/PLACEHOLDER/thee/harness/internal/db"
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
	History  []provider.Message
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
	OnAssistant         func(text string)
	OnTool              func(ToolActivity)
	AskUser             func(ctx context.Context, question string, choices []string) (string, error)
	ForceEscalate       bool
	Interactive         bool
	History             []provider.Message
	// replaceTools overwrites tools of the same name after registration.
	replaceTools []tools.Tool
}

// ToolActivity describes one tool call for live display. OnTool receives it
// once before the tool runs (Done false) and once after (Done true). It is
// never written to the trace, so arguments and output stay out of ZDR logs.
type ToolActivity struct {
	Name   string
	Args   json.RawMessage
	Done   bool
	OK     bool
	Output string
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

	database, err := db.Open(ws.Root())
	if err != nil {
		return Outcome{State: StateFailed, Reason: "setup", ExitCode: 3}, err
	}
	defer database.Close()
	warnDB(database.StartChat(db.Chat{
		ID: sid, Workspace: ws.Root(), Model: model.Model, Provider: model.Provider,
	}))

	session := &tools.SessionState{}
	var totals observability.UsageTotals
	var malformed int
	var loopsDetected int
	history := append([]provider.Message(nil), cfg.History...)
	if goal := strings.TrimSpace(cfg.Goal); goal != "" {
		history = append(history, provider.Message{Role: "user", Content: goal})
		warnDB(database.AppendMessages(sid, []db.Message{{Role: "user", Content: goal}}))
	}
	var lastTurnID int64
	appendMsg := func(msg provider.Message) {
		warnDB(database.AppendMessages(sid, []db.Message{storedMessage(msg)}))
	}

	emit := func(ev observability.Event) {
		_ = tw.Append(ev)
		if cfg.OnEvent != nil {
			cfg.OnEvent(ev)
		}
		if kind, ok := eventKind(ev); ok {
			payload, err := json.Marshal(ev)
			if err != nil {
				warnDB(err)
				return
			}
			warnDB(database.RecordEvent(sid, db.Event{Kind: kind, Payload: payload}))
		}
	}
	finish := func(o Outcome) Outcome {
		o.History = append([]provider.Message(nil), history...)
		emit(observability.SessionEndEvent(o.State, o.Reason, totals, session.ToolRounds, session.VerifyAttempts, session.EscalationsUsed, malformed, loopsDetected))
		warnDB(database.FinishChat(sid, db.Summary{
			Outcome:         o.State,
			Reason:          o.Reason,
			InputTokens:     totals.InputTokens,
			OutputTokens:    totals.OutputTokens,
			CachedTokens:    totals.CachedTokens,
			ReasoningTokens: totals.ReasoningTokens,
			CostUSD:         totals.CostUSD,
		}))
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
	for _, extra := range cfg.replaceTools {
		reg.Register(extra)
	}

	toolEnv := &tools.Env{
		Workspace:      ws,
		Trusted:        trusted,
		Session:        session,
		Headless:       !cfg.Interactive,
		AllowGitCommit: eff.AllowCommit,
		FilterCommand:  eff.FilterCommand,
		AskUser:        cfg.AskUser,
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

	compactHistory := func(msgs []provider.Message, budget, window int) []provider.Message {
		next := contextengine.CompactHistory(msgs, budget, window, estimator)
		if historyChanged(msgs, next) {
			payload, err := json.Marshal(map[string]int{
				"tokens_before": estimator.EstimateMessages(msgs),
				"tokens_after":  estimator.EstimateMessages(next),
			})
			if err != nil {
				warnDB(err)
			} else {
				warnDB(database.RecordEvent(sid, db.Event{Kind: "compact", Payload: payload}))
			}
		}
		return next
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
			window := model.Capabilities.ContextWindow
			history = compactHistory(history, contextengine.TurnInputBudget(window), window)
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
					history = compactHistory(history, turn.EstimatedInputTokens/2, model.Capabilities.ContextWindow)
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
			started := time.Now()
			out, resp, err := toolcall.Next(ctx, activeProv, req, toolcall.DispatchInput{
				ToolProtocol: model.Capabilities.ToolProtocol,
				Phase:        phase,
				Specs:        reg.Specs(),
			})
			callCost, _ := router.CostUSD(rates, resp.Usage)
			if id, werr := database.RecordTurn(sid, db.Turn{
				Request:              storedRequest(turn),
				Response:             storedResponse(resp, err),
				InputTokens:          resp.Usage.InputTokens,
				OutputTokens:         resp.Usage.OutputTokens,
				CachedTokens:         resp.Usage.CachedTokens,
				ReasoningTokens:      resp.Usage.ReasoningTokens,
				CostUSD:              callCost,
				Latency:              time.Since(started),
				PrefixHash:           turn.PrefixHash,
				EstimatedInputTokens: turn.EstimatedInputTokens,
			}); werr != nil {
				warnDB(werr)
			} else {
				lastTurnID = id
			}
			if err != nil && cfg.Interactive && errors.Is(err, toolcall.ErrNoToolCalls) {
				if text := visibleAssistantText(out.Text); text != "" {
					budget.Charge(resp.Usage.InputTokens, resp.Usage.OutputTokens)
					budget.ChargeCost(callCost)
					totals.Add(resp.Usage, callCost)
					emit(observability.ModelResponseEvent(callCost, resp.Usage))
					if cfg.OnAssistant != nil {
						cfg.OnAssistant(text)
					}
					msg := provider.Message{Role: "assistant", Content: text}
					history = append(history, msg)
					appendMsg(msg)
					emit(observability.StateTransitionEvent(state, StateDone, "answered"))
					return finish(Outcome{State: StateDone, Reason: "answered", ExitCode: 0}), nil
				}
			}
			if err != nil && !repairableToolErr(err) {
				return finish(Outcome{State: StateFailed, Reason: "model_error", ExitCode: 1}), err
			}
			if err != nil {
				malformed++
				if repair.ConsecutiveFailure(err) {
					if tryEscalate(router.EscalationInput{ConsecutiveMalformed: true}) {
						repair = RepairState{}
						continue
					}
					return finish(Outcome{State: StateFailed, Reason: "tool_parse_exhausted", ExitCode: 1}), err
				}
				if repair.ShouldRepair(err) {
					msg := repair.RepairMessage()
					history = append(history, msg)
					appendMsg(msg)
					continue
				}
				return finish(Outcome{State: StateFailed, Reason: "no_tool_call", ExitCode: 1}), err
			}
			repair = RepairState{}
			text := visibleAssistantText(out.Text)
			if text != "" && cfg.OnAssistant != nil {
				cfg.OnAssistant(text)
			}
			estimator.ObserveUsage(resp.Usage.InputTokens, turn.EstimatedInputTokens)
			budget.Charge(resp.Usage.InputTokens, resp.Usage.OutputTokens)
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
				msg := provider.Message{Role: "user", Content: nudge}
				history = append(history, msg)
				appendMsg(msg)
			}
			history = contextengine.AppendAssistantToolCalls(history, out.Calls)
			if text != "" && len(history) > 0 {
				history[len(history)-1].Content = text
			}
			if n := len(history); n > 0 && len(resp.Replay) > 0 {
				history[n-1].Replay = resp.Replay
			}
			if n := len(history); n > 0 {
				appendMsg(history[n-1])
			}
			emit(observability.StateTransitionEvent(state, StateTooling, ""))
			state = StateTooling
			session.ToolRounds++

			noteTool := func(call provider.ToolCall, res tools.Result) {
				emit(observability.ToolResultEvent(res.OK))
				if cfg.OnTool != nil {
					cfg.OnTool(ToolActivity{Name: call.Name, Args: call.Arguments, Done: true, OK: res.OK, Output: res.Text})
				}
				history = contextengine.AppendToolResult(history, call.ID, call.Name, res.Text)
				appendMsg(provider.Message{Role: "tool", Content: res.Text, ToolCallID: call.ID, Name: call.Name})
				errText := ""
				if !res.OK {
					errText = res.Text
				}
				warnDB(database.RecordTool(sid, db.ToolCall{
					TurnID:    lastTurnID,
					Name:      call.Name,
					Phase:     phase,
					Arguments: call.Arguments,
					Result:    res.Text,
					OK:        res.OK,
					Error:     errText,
				}))
			}
			rejectCall := func(call provider.ToolCall, msg string) {
				if cfg.OnTool != nil {
					cfg.OnTool(ToolActivity{Name: call.Name, Args: call.Arguments})
				}
				noteTool(call, tools.Fail(toolEnv, call.Name, msg))
				emit(observability.StateTransitionEvent(state, StateActing, ""))
				state = StateActing
			}

			for i, call := range out.Calls {
				if out.Salvaged {
					emit(observability.ToolCallSalvagedEvent(call.Name))
				} else {
					emit(observability.ToolCallEvent(call.Name))
				}
				loops.RecordToolCall(call.Name, call.Arguments)
				if err := reg.Check(call.Name, phase); err != nil {
					rejectCall(call, err.Error())
					continue
				}
				tool, err := reg.Get(call.Name)
				if err != nil {
					rejectCall(call, err.Error())
					continue
				}
				if cfg.OnTool != nil {
					cfg.OnTool(ToolActivity{Name: call.Name, Args: call.Arguments})
				}
				res, err := tool.Run(ctx, toolEnv, call.Arguments)
				if err != nil {
					noteTool(call, tools.Fail(toolEnv, call.Name, err.Error()))
					for _, later := range out.Calls[i+1:] {
						emit(observability.ToolCallEvent(later.Name))
						if cfg.OnTool != nil {
							cfg.OnTool(ToolActivity{Name: later.Name, Args: later.Arguments})
						}
						noteTool(later, tools.Fail(toolEnv, later.Name, "not run"))
					}
					return finish(Outcome{State: StateFailed, Reason: "tool_error", ExitCode: 1}), err
				}
				noteTool(call, res)
				if call.Name == "patch" && res.OK {
					var pa struct {
						Path string `json:"path"`
					}
					_ = json.Unmarshal(call.Arguments, &pa)
					if pa.Path != "" {
						_ = cpStore.Save(pa.Path)
					}
				}
				if call.Name == "finish" && res.OK && session.FinishSummary != "" && cfg.OnAssistant != nil {
					cfg.OnAssistant(session.FinishSummary)
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
					appendMsg(history[len(history)-1])
					state = StateActing
					continue
				}
				bundle := verifier.ParseGoTest(vres.Excerpt)
				history = appendVerifyFailure(history, verifier.FormatVerifyFailure(vres, bundle))
				appendMsg(history[len(history)-1])
				emit(observability.StateTransitionEvent(state, StateActing, "verify_fail"))
				state = StateActing
			}

		default:
			return finish(Outcome{State: StateFailed, Reason: "bad_state", ExitCode: 1}), nil
		}
	}
	if state == StateDone {
		reason := "no_changes"
		if session.EverVerifyPassed {
			reason = "verified"
		}
		return finish(Outcome{State: StateDone, Reason: reason, ExitCode: 0}), nil
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

func visibleAssistantText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return ""
	}
	return text
}
