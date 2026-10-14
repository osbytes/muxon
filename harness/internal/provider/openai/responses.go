package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

const openCodeGoResponsesURL = "https://opencode.ai/inference/go/v1/responses"

// responsesModels are OpenCode Go models that speak the Responses API.
var responsesModels = map[string]bool{
	"gpt-6-luna":                 true,
	"gpt-5.6-luna":               true,
	"grok-4.7":                   true,
	"grok-4.6":                   true,
	"muse-spark-1.3-contributor": true,
	"muse-spark-1.2-contributor": true,
}

func responsesModel(id string) bool {
	return responsesModels[id]
}

func (c *Client) responsesEndpoint() string {
	base := strings.TrimRight(c.BaseURL, "/")
	if strings.Contains(base, "opencode.ai") && strings.Contains(base, "/go/") {
		return openCodeGoResponsesURL
	}
	return base + "/responses"
}

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type responsesHTTP struct {
	Model           string          `json:"model"`
	Instructions    string          `json:"instructions,omitempty"`
	Input           []any           `json:"input"`
	Tools           []responsesTool `json:"tools,omitempty"`
	MaxOutputTokens int             `json:"max_output_tokens,omitempty"`
	Store           bool            `json:"store"`
}

func (c *Client) completeResponses(ctx context.Context, req provider.Request) (provider.Response, error) {
	body, err := encodeResponses(req)
	if err != nil {
		return provider.Response{}, err
	}
	var out provider.Response
	err = withRetry(ctx, func() error {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.responsesEndpoint(), bytes.NewReader(body))
		if err != nil {
			return err
		}
		c.authorize(httpReq)
		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("responses: %w", err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return fmt.Errorf("responses: %w", err)
		}
		if resp.StatusCode >= 400 {
			return classifyHTTPError(resp.StatusCode, string(b))
		}
		decoded, err := decodeResponses(b)
		if err != nil {
			return err
		}
		out = decoded
		return nil
	})
	return out, err
}

func encodeResponses(req provider.Request) ([]byte, error) {
	var instructions []string
	var input []any
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			if text := strings.TrimSpace(msg.Content); text != "" {
				instructions = append(instructions, text)
			}
		case "assistant":
			if len(msg.Replay) > 0 {
				var items []any
				if err := json.Unmarshal(msg.Replay, &items); err != nil {
					return nil, fmt.Errorf("responses replay: %w", err)
				}
				input = append(input, items...)
				continue
			}
			if msg.Content != "" {
				input = append(input, map[string]string{"role": "assistant", "content": msg.Content})
			}
			for _, call := range msg.ToolCalls {
				args := string(call.Arguments)
				if args == "" {
					args = "{}"
				}
				input = append(input, map[string]string{
					"type":      "function_call",
					"call_id":   call.ID,
					"name":      call.Name,
					"arguments": args,
				})
			}
		case "tool":
			input = append(input, map[string]string{
				"type":    "function_call_output",
				"call_id": msg.ToolCallID,
				"output":  msg.Content,
			})
		default:
			input = append(input, map[string]string{"role": "user", "content": msg.Content})
		}
	}
	if len(input) == 0 {
		input = append(input, map[string]string{"role": "user", "content": ""})
	}
	var tools []responsesTool
	for _, tool := range req.Tools {
		params := tool.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		tools = append(tools, responsesTool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		})
	}
	wire := responsesHTTP{
		Model:           req.ModelID,
		Instructions:    strings.Join(instructions, "\n\n"),
		Input:           input,
		Tools:           tools,
		MaxOutputTokens: req.MaxOutputTokens,
		Store:           false,
	}
	return json.Marshal(wire)
}

type responsesBody struct {
	Output []json.RawMessage `json:"output"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		InputDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		OutputDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
}

func decodeResponses(body []byte) (provider.Response, error) {
	var wire responsesBody
	if err := json.Unmarshal(body, &wire); err != nil {
		return provider.Response{}, fmt.Errorf("responses: %w", err)
	}
	if wire.Error != nil && wire.Error.Message != "" {
		return provider.Response{}, fmt.Errorf("responses: %s", wire.Error.Message)
	}
	var out provider.Response
	var text []string
	for _, raw := range wire.Output {
		var item struct {
			Type      string          `json:"type"`
			CallID    string          `json:"call_id"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Arguments string          `json:"arguments"`
			Content   json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return provider.Response{}, fmt.Errorf("responses output: %w", err)
		}
		switch item.Type {
		case "message", "":
			if line := outputText(item.Content); line != "" {
				text = append(text, line)
			}
		case "function_call":
			id := item.CallID
			if id == "" {
				id = item.ID
			}
			args := item.Arguments
			if args == "" {
				args = "{}"
			}
			out.ToolCalls = append(out.ToolCalls, provider.ToolCall{
				ID:        id,
				Name:      item.Name,
				Arguments: json.RawMessage(args),
			})
		}
	}
	out.Text = strings.Join(text, "")
	if len(out.ToolCalls) > 0 {
		out.FinishReason = "tool_calls"
	} else if out.Text != "" {
		out.FinishReason = "stop"
	}
	if len(wire.Output) > 0 {
		out.Replay = json.RawMessage(appendJSONArray(wire.Output))
	}
	out.Usage = provider.Usage{
		InputTokens:     wire.Usage.InputTokens,
		OutputTokens:    wire.Usage.OutputTokens,
		CachedTokens:    wire.Usage.InputDetails.CachedTokens,
		ReasoningTokens: wire.Usage.OutputDetails.ReasoningTokens,
	}
	return out, nil
}

func outputText(content json.RawMessage) string {
	if len(content) == 0 || string(content) == "null" {
		return ""
	}
	var plain string
	if err := json.Unmarshal(content, &plain); err == nil {
		return plain
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &parts); err != nil {
		return ""
	}
	var b strings.Builder
	for _, part := range parts {
		if part.Type == "" || part.Type == "output_text" || part.Type == "text" {
			b.WriteString(part.Text)
		}
	}
	return b.String()
}

func appendJSONArray(items []json.RawMessage) []byte {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(item)
	}
	b.WriteByte(']')
	return b.Bytes()
}

type singleResponse struct {
	resp provider.Response
	done bool
}

func (s *singleResponse) Next() (provider.Response, error) {
	if s.done {
		return provider.Response{}, fmt.Errorf("stream closed")
	}
	s.done = true
	return s.resp, nil
}

func (s *singleResponse) Close() error {
	s.done = true
	return nil
}
