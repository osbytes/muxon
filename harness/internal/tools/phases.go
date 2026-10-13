package tools

import "fmt"

// Phase names for tool gating.
const (
	PhasePlanning = "planning"
	PhaseEditing  = "editing"
)

var planningTools = map[string]bool{
	"search":   true,
	"list":     true,
	"read":     true,
	"repo-map": true,
}

var editingTools = map[string]bool{
	"read":     true,
	"search":   true,
	"patch":    true,
	"write":    true,
	"delete":   true,
	"finish":   true,
	"repo-map": true,
}

// Check returns an error when the tool is not allowed in the phase.
func (r *Registry) Check(name, phase string) error {
	switch phase {
	case PhasePlanning:
		if !planningTools[name] {
			return fmt.Errorf("tool %q not allowed in planning phase", name)
		}
	case PhaseEditing:
		if !editingTools[name] {
			return fmt.Errorf("tool %q not allowed in editing phase", name)
		}
	default:
		return fmt.Errorf("unknown phase %q", phase)
	}
	return nil
}
