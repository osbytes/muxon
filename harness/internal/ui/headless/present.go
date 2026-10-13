package headless

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/PLACEHOLDER/thee/harness/internal/observability"
)

// Mode selects CLI output style.
type Mode int

const (
	ModeHuman Mode = 0
	ModeJSON  Mode = 1
	ModeQuiet Mode = 2
)

var ndjsonTypes = map[string]bool{
	"state.transition": true,
	"tool.call":        true,
	"tool.result":      true,
	"verify.run":       true,
	"session.end":      true,
}

// Presenter writes harness events for headless runs.
type Presenter struct {
	Mode   Mode
	Out    io.Writer
	Quiet  bool
	Reason string
	State  string
}

// NewPresenter builds a presenter for the given mode.
func NewPresenter(mode Mode) *Presenter {
	return &Presenter{Mode: mode, Out: os.Stdout}
}

// Handle processes one trace event for output.
func (p *Presenter) Handle(ev observability.Event) {
	switch p.Mode {
	case ModeJSON:
		if ndjsonTypes[ev.Type] {
			b, _ := json.Marshal(ev)
			fmt.Fprintln(p.Out, string(b))
		}
	case ModeQuiet:
		if ev.Type == "session.end" {
			p.State = ev.Outcome
			p.Reason = ev.Reason
		}
	}
}

// FinalLine returns the quiet-mode summary line.
func (p *Presenter) FinalLine() string {
	if p.State == "done" {
		return "done"
	}
	if p.Reason != "" {
		return fmt.Sprintf("failed: %s", p.Reason)
	}
	return "failed"
}
