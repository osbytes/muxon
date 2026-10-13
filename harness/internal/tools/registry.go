package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Result is returned to the model from a tool.
type Result struct {
	OK   bool
	Text string
}

// WorkspaceOps is implemented by workspace.Workspace.
type WorkspaceOps interface {
	Resolve(string) (string, error)
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte) error
	CheckFresh(string) error
	NoteRead(string, []byte)
	Changed() bool
}

// Env is per-run tool execution context.
type Env struct {
	Workspace WorkspaceOps
	Trusted   bool
	Session   *SessionState
}

// SessionState holds cross-tool session flags.
type SessionState struct {
	PendingVerify bool
	FinishSummary string
}

// Tool implements one harness tool.
type Tool interface {
	Name() string
	Spec() Spec
	Run(ctx context.Context, env *Env, args json.RawMessage) (Result, error)
}

// Spec describes a tool for the model.
type Spec struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// Registry holds registered tools.
type Registry struct {
	byName map[string]Tool
	order  []string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Tool)}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	name := t.Name()
	if _, ok := r.byName[name]; !ok {
		r.order = append(r.order, name)
	}
	r.byName[name] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	return t, nil
}

// Names returns registered tool names in order.
func (r *Registry) Names() []string {
	return append([]string(nil), r.order...)
}

// Specs returns tool specs for native protocol.
func (r *Registry) Specs() []Spec {
	out := make([]Spec, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.byName[n].Spec())
	}
	return out
}
