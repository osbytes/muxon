package contextengine

import (
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

// AppendToolResult adds a tool result user message.
func AppendToolResult(msgs []provider.Message, callID, name, text string) []provider.Message {
	msgs = append(msgs, provider.Message{
		Role:       "tool",
		ToolCallID: callID,
		Name:       name,
		Content:    text,
	})
	return msgs
}

// AppendAssistantToolCalls records assistant tool calls in history.
func AppendAssistantToolCalls(msgs []provider.Message, calls []provider.ToolCall) []provider.Message {
	msgs = append(msgs, provider.Message{
		Role:      "assistant",
		ToolCalls: calls,
	})
	return msgs
}
