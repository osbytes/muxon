package director

import (
	"encoding/json"
	"log/slog"

	"github.com/PLACEHOLDER/thee/harness/internal/contextengine"
	"github.com/PLACEHOLDER/thee/harness/internal/db"
	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func warnDB(err error) {
	if err != nil {
		slog.Error("database write", "err", err)
	}
}

func eventKind(ev observability.Event) (string, bool) {
	switch ev.Type {
	case "state.transition":
		return "state", true
	case "loop.detected":
		return "loop", true
	case "verify.run":
		return "verify", true
	case "goal.start":
		return "goal", true
	default:
		return "", false
	}
}

func historyChanged(before, after []provider.Message) bool {
	if len(before) != len(after) {
		return true
	}
	for i := range before {
		if before[i].Role != after[i].Role || before[i].Content != after[i].Content {
			return true
		}
	}
	return false
}

func storedMessage(msg provider.Message) db.Message {
	var calls json.RawMessage
	if len(msg.ToolCalls) > 0 {
		calls, _ = json.Marshal(msg.ToolCalls)
	}
	return db.Message{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
		Name:       msg.Name,
		ToolCalls:  calls,
	}
}

func storedRequest(turn contextengine.TurnOutput) json.RawMessage {
	raw, err := json.Marshal(struct {
		Messages []provider.Message `json:"messages"`
		Tools    []provider.ToolDef `json:"tools,omitempty"`
	}{Messages: turn.Messages, Tools: turn.Tools})
	if err != nil || len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func storedResponse(resp provider.Response, callErr error) json.RawMessage {
	body := struct {
		Text         string              `json:"text,omitempty"`
		ToolCalls    []provider.ToolCall `json:"tool_calls,omitempty"`
		FinishReason string              `json:"finish_reason,omitempty"`
		Usage        provider.Usage      `json:"usage"`
		Error        string              `json:"error,omitempty"`
	}{
		Text:         resp.Text,
		ToolCalls:    resp.ToolCalls,
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
	}
	if callErr != nil {
		body.Error = callErr.Error()
	}
	raw, err := json.Marshal(body)
	if err != nil || len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
