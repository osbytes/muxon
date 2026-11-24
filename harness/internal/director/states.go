package director

import (
	"github.com/PLACEHOLDER/muxon/harness/internal/observability"
)

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

// transition records a state change on the trace.
func transition(tw *observability.Writer, from, to, reason string) {
	_ = tw.Append(observability.StateTransitionEvent(from, to, reason))
}
