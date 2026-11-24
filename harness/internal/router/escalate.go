package router

// EscalationInput collects counters for escalation decisions.
type EscalationInput struct {
	EscalationGeneration   int
	MaxEscalations         int
	ConsecutiveVerifyFails int
	EscalateAfterVerify    int
	ToolRounds             int
	MaxToolRounds          int
	VerifyPassed           bool
	ConsecutiveMalformed   bool
	LoopSecondFire         bool
	LowRouterConfidence    bool
	TagMatchAlwaysEscalate bool
	ForceUser              bool
	RemainingCostUSD       float64
	EstimatedNextCostUSD   float64
	HasEscalatedRole       bool
}

// EscalationDecision is whether to escalate and why.
type EscalationDecision struct {
	Escalate bool
	Reason   string
}

// ShouldEscalate applies spec escalation triggers.
func ShouldEscalate(in EscalationInput) EscalationDecision { //nolint:gocyclo // flat list of escalation triggers
	if !in.HasEscalatedRole {
		return EscalationDecision{}
	}
	if in.EscalationGeneration >= in.MaxEscalations {
		return EscalationDecision{}
	}
	if in.EscalateAfterVerify <= 0 {
		in.EscalateAfterVerify = 2
	}
	if in.MaxEscalations <= 0 {
		in.MaxEscalations = 1
	}
	if in.RemainingCostUSD > 0 && in.EstimatedNextCostUSD > in.RemainingCostUSD {
		return EscalationDecision{Reason: "budget_skip"}
	}
	if in.ForceUser {
		return EscalationDecision{Escalate: true, Reason: "user"}
	}
	if in.ConsecutiveVerifyFails >= in.EscalateAfterVerify {
		return EscalationDecision{Escalate: true, Reason: "verify_fails"}
	}
	if in.MaxToolRounds > 0 && !in.VerifyPassed && in.ToolRounds >= (in.MaxToolRounds*60)/100 {
		return EscalationDecision{Escalate: true, Reason: "tool_rounds_stall"}
	}
	if in.ConsecutiveMalformed {
		return EscalationDecision{Escalate: true, Reason: "tool_parse"}
	}
	if in.LoopSecondFire {
		return EscalationDecision{Escalate: true, Reason: "loop"}
	}
	if in.LowRouterConfidence {
		return EscalationDecision{Escalate: true, Reason: "low_confidence"}
	}
	if in.TagMatchAlwaysEscalate {
		return EscalationDecision{Escalate: true, Reason: "policy_tag"}
	}
	return EscalationDecision{}
}
