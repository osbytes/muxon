package director

// Director states recorded in state.transition events.
const (
	StateIdle       = "idle"
	StateActing     = "acting"
	StateTooling    = "tooling"
	StateVerifying  = "verifying"
	StateBlocked    = "blocked"
	StateEscalating = "escalating"
	StateDone       = "done"
	StateFailed     = "failed"
	StatePaused     = "paused"
	StateCancelled  = "cancelled"
)
