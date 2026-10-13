package observability

const SchemaVersion = "1.0.0"

// Event is one JSONL trace record. Fields depend on Type.
type Event struct {
	Type string `json:"type"`

	SchemaVersion string `json:"schema_version,omitempty"`
	TheeVersion   string `json:"thee_version,omitempty"`
	Workspace     string `json:"workspace,omitempty"`

	Outcome string `json:"outcome,omitempty"`
	Reason  string `json:"reason,omitempty"`

	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`

	Name string `json:"name,omitempty"`
	OK   bool   `json:"ok,omitempty"`

	Command    []string `json:"command,omitempty"`
	ExitCode   int      `json:"exit_code,omitempty"`
	Passed     bool     `json:"passed,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
}

func SessionStartEvent(theeVersion, workspace string) Event {
	return Event{
		Type:          "session.start",
		SchemaVersion: SchemaVersion,
		TheeVersion:   theeVersion,
		Workspace:     workspace,
	}
}

func SessionEndEvent(outcome, reason string) Event {
	return Event{
		Type:    "session.end",
		Outcome: outcome,
		Reason:  reason,
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

func ToolResultEvent(ok bool) Event {
	return Event{Type: "tool.result", OK: ok}
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
