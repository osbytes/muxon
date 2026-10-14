package director

import (
	"errors"
	"fmt"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/toolcall"
)

func repairableToolErr(err error) bool {
	return errors.Is(err, toolcall.ErrNoToolCalls) || errors.Is(err, toolcall.ErrMultipleToolCalls)
}

// RepairState tracks one repair attempt per parse streak.
type RepairState struct {
	Pending bool
	Used    bool
	LastErr string
}

// ShouldRepair returns true when a repair turn is allowed.
func (r *RepairState) ShouldRepair(err error) bool {
	if err == nil {
		r.Pending = false
		r.LastErr = ""
		return false
	}
	if r.Used {
		return false
	}
	r.Pending = true
	r.LastErr = err.Error()
	return true
}

// RepairMessage builds the user message for a repair turn.
func (r *RepairState) RepairMessage() provider.Message {
	r.Used = true
	return provider.Message{
		Role:    "user",
		Content: fmt.Sprintf("Tool call schema error: %s. Emit one valid tool call.", r.LastErr),
	}
}

// ConsecutiveFailure is true after repair was used and still failing.
func (r *RepairState) ConsecutiveFailure(err error) bool {
	return err != nil && r.Used
}
