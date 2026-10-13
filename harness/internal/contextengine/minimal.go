package contextengine

import (
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

// TurnInput builds one model turn.
type TurnInput struct {
	Goal         string
	History      []provider.Message
	ToolProtocol string
	ToolSpecs    []tools.Spec
}

// TurnOutput is assembled messages and optional native tools.
type TurnOutput struct {
	Messages             []provider.Message
	Tools                []provider.ToolDef
	PrefixHash           string
	EstimatedInputTokens int
}

// BuildTurn assembles a minimal native-protocol turn.
func BuildTurn(in TurnInput) TurnOutput {
	sys := "You are thee, a coding agent. Use tools to fix the goal."
	msgs := []provider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: in.Goal},
	}
	msgs = append(msgs, in.History...)
	out := TurnOutput{
		Messages:             msgs,
		PrefixHash:           hashPrefix(sys, in.Goal),
		EstimatedInputTokens: estimateTokens(msgs),
	}
	if in.ToolProtocol == "native" {
		for _, s := range in.ToolSpecs {
			out.Tools = append(out.Tools, provider.ToolDef{
				Name:        s.Name,
				Description: s.Description,
				Parameters:  s.Parameters,
			})
		}
	}
	return out
}

func hashPrefix(sys, goal string) string {
	return sys + "|" + goal
}

func estimateTokens(msgs []provider.Message) int {
	n := 0
	for _, m := range msgs {
		n += len(m.Content) / 4
	}
	return n
}

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
