package provider

import (
	"context"
	"encoding/json"
)

// Provider calls a model backend.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (Stream, error)
}

// Request is a normalized completion request.
type Request struct {
	ModelID         string
	Messages        []Message
	Tools           []ToolDef
	MaxOutputTokens int
}

// Message is a chat message.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
	Name       string
}

// ToolDef describes a tool for native protocol.
type ToolDef struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ToolCall is a model-emitted tool invocation.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Response is a normalized completion response.
type Response struct {
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
}

// Usage is normalized token usage.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Stream delivers completion chunks.
type Stream interface {
	Next() (Response, error)
	Close() error
}

// APIErrorKind classifies provider errors.
type APIErrorKind string

const (
	APIErrorToolsRejected      APIErrorKind = "tools_rejected"
	APIErrorConstraintRejected APIErrorKind = "constraint_rejected"
	APIErrorContextLength      APIErrorKind = "context_length"
)

// APIError is a typed provider failure.
type APIError struct {
	Kind       APIErrorKind
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}
