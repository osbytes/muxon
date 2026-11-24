package director

import (
	"encoding/json"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/PLACEHOLDER/muxon/harness/internal/contextengine"
	"github.com/PLACEHOLDER/muxon/harness/internal/db"
	"github.com/PLACEHOLDER/muxon/harness/internal/observability"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
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

func recordChatLog(database *db.DB, chatID, level, message, source, errText string) {
	warnDB(database.RecordLog(chatID, db.Log{
		Level:   level,
		Message: message,
		Source:  source,
		Error:   clipLog(errText),
	}))
}

// toolFailureText is the stored reason for a failed model or tool-call turn.
func toolFailureText(err error, resp provider.Response) string {
	var b strings.Builder
	if err != nil {
		b.WriteString(err.Error())
	}
	if names := toolNames(resp.ToolCalls); names != "" {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("calls: ")
		b.WriteString(names)
	}
	if text := strings.TrimSpace(resp.Text); text != "" {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("model text: ")
		b.WriteString(text)
	}
	return b.String()
}

func toolNames(calls []provider.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		if call.Name != "" {
			names = append(names, call.Name)
		}
	}
	return strings.Join(names, ", ")
}

func clipLog(s string) string {
	const max = 4096
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	if cut == 0 {
		cut = max
	}
	return s[:cut] + "…"
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
