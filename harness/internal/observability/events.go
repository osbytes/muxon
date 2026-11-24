package observability

import (
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

const SchemaVersion = "1.0.0"

// Event is one JSONL trace record. Fields depend on Type.
type Event struct {
	Type string `json:"type"`

	SchemaVersion string `json:"schema_version,omitempty"`
	MuxonVersion   string `json:"muxon_version,omitempty"`
	Workspace     string `json:"workspace,omitempty"`

	Outcome string `json:"outcome,omitempty"`
	Reason  string `json:"reason,omitempty"`

	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`

	Name          string `json:"name,omitempty"`
	OK            bool   `json:"ok,omitempty"`
	Salvaged      bool   `json:"salvaged,omitempty"`
	ApplyStrategy string `json:"apply_strategy,omitempty"`

	Command    []string `json:"command,omitempty"`
	ExitCode   int      `json:"exit_code,omitempty"`
	Passed     bool     `json:"passed,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`

	TaskType         string  `json:"task_type,omitempty"`
	RouterConfidence float64 `json:"router_confidence,omitempty"`
	GoalID           string  `json:"goal_id,omitempty"`

	FromModel            string `json:"from_model,omitempty"`
	ToModel              string `json:"to_model,omitempty"`
	EscalationGeneration int    `json:"escalation_generation,omitempty"`

	InputTokens        int     `json:"input_tokens,omitempty"`
	OutputTokens       int     `json:"output_tokens,omitempty"`
	CachedTokens       int     `json:"cached_tokens,omitempty"`
	ReasoningTokens    int     `json:"reasoning_tokens,omitempty"`
	CostUSD            float64 `json:"cost_usd,omitempty"`
	TotalCostUSD       float64 `json:"total_cost_usd,omitempty"`
	ToolRounds         int     `json:"tool_rounds,omitempty"`
	VerifyAttempts     int     `json:"verify_attempts,omitempty"`
	MalformedToolCalls int     `json:"malformed_tool_calls,omitempty"`
	LoopsDetected      int     `json:"loops_detected,omitempty"`
}

// UsageTotals aggregates session usage for session.end.
type UsageTotals struct {
	InputTokens     int
	OutputTokens    int
	CachedTokens    int
	ReasoningTokens int
	CostUSD         float64
}

// Add merges one model call into totals.
func (t *UsageTotals) Add(u provider.Usage, costUSD float64) {
	t.InputTokens += u.InputTokens
	t.OutputTokens += u.OutputTokens
	t.CachedTokens += u.CachedTokens
	t.ReasoningTokens += u.ReasoningTokens
	if costUSD > 0 {
		t.CostUSD += costUSD
	}
}

func SessionStartEvent(muxonVersion, workspace string) Event {
	return Event{
		Type:          "session.start",
		SchemaVersion: SchemaVersion,
		MuxonVersion:   muxonVersion,
		Workspace:     workspace,
	}
}

func GoalStartEvent(taskType string, confidence float64) Event {
	return Event{
		Type:             "goal.start",
		TaskType:         taskType,
		RouterConfidence: confidence,
		GoalID:           "1",
	}
}

func SessionEndEvent(outcome, reason string, totals UsageTotals, toolRounds, verifyAttempts, escalations, malformed, loops int) Event {
	return Event{
		Type:                 "session.end",
		Outcome:              outcome,
		Reason:               reason,
		TotalCostUSD:         totals.CostUSD,
		InputTokens:          totals.InputTokens,
		OutputTokens:         totals.OutputTokens,
		CachedTokens:         totals.CachedTokens,
		ReasoningTokens:      totals.ReasoningTokens,
		ToolRounds:           toolRounds,
		VerifyAttempts:       verifyAttempts,
		EscalationGeneration: escalations,
		MalformedToolCalls:   malformed,
		LoopsDetected:        loops,
	}
}

func StateTransitionEvent(from, to, reason string) Event {
	return Event{
		Type:   "state.transition",
		From:   from,
		To:     to,
		Reason: reason,
	}
}

func ToolCallEvent(name string) Event {
	return Event{Type: "tool.call", Name: name}
}

func ToolCallSalvagedEvent(name string) Event {
	return Event{Type: "tool.call", Name: name, Salvaged: true}
}

func TruncatedByServerEvent() Event {
	return Event{Type: "truncated_by_server"}
}

func ToolResultEvent(ok bool, applyStrategy string) Event {
	return Event{Type: "tool.result", OK: ok, ApplyStrategy: applyStrategy}
}

func VerifyRunEvent(cmd []string, exitCode int, passed bool, durationMS int64) Event {
	return Event{
		Type:       "verify.run",
		Command:    cmd,
		ExitCode:   exitCode,
		Passed:     passed,
		DurationMS: durationMS,
	}
}

func ModelResponseEvent(costUSD float64, u provider.Usage) Event {
	return Event{
		Type:            "model.response",
		CostUSD:         costUSD,
		InputTokens:     u.InputTokens,
		OutputTokens:    u.OutputTokens,
		CachedTokens:    u.CachedTokens,
		ReasoningTokens: u.ReasoningTokens,
	}
}

func EscalateEvent(fromModel, toModel, reason string, generation int) Event {
	return Event{
		Type:                 "escalate",
		FromModel:            fromModel,
		ToModel:              toModel,
		Reason:               reason,
		EscalationGeneration: generation,
	}
}
