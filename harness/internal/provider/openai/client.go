package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// Client is an OpenAI-compatible chat completions client.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Headers    map[string]string
}

// NewClient returns a client with sane defaults.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type chatRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	Stream         bool                `json:"stream"`
	Tools          []toolWire          `json:"tools,omitempty"`
	ResponseFormat *responseFormatWire `json:"response_format,omitempty"`
}

type responseFormatWire struct {
	Type       string          `json:"type"`
	JSONSchema *jsonSchemaWire `json:"json_schema,omitempty"`
}

type jsonSchemaWire struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []toolCallWire `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

type toolWire struct {
	Type     string     `json:"type"`
	Function toolFnWire `json:"function"`
}

type toolFnWire struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type toolCallWire struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Function toolCallFn `json:"function"`
}

type toolCallFn struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   string         `json:"content"`
			ToolCalls []toolCallWire `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage usageWire `json:"usage"`
}

type usageWire struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
	PromptDetails    struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

// Complete implements provider.Provider.
func (c *Client) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	if responsesModel(req.ModelID) {
		return c.completeResponses(ctx, req)
	}
	out, err := c.completeOnce(ctx, req)
	if err != nil {
		if api, ok := err.(*provider.APIError); ok && api.Kind == provider.APIErrorConstraintRejected {
			if req.ResponseFormat.Kind == provider.FormatJSONSchema {
				req.ResponseFormat = provider.ResponseFormat{Kind: provider.FormatJSONObject}
				return c.completeOnce(ctx, req)
			}
		}
	}
	return out, err
}

func (c *Client) completeOnce(ctx context.Context, req provider.Request) (provider.Response, error) {
	body, err := c.encodeRequest(req, false)
	if err != nil {
		return provider.Response{}, err
	}
	var out provider.Response
	err = withRetry(ctx, func() error {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return err
		}
		c.authorize(httpReq)
		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(resp.Body)
			return classifyHTTPError(resp.StatusCode, string(b))
		}
		var cr chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
			return err
		}
		out = decodeResponse(cr)
		return nil
	})
	return out, err
}

// Stream implements provider.Provider.
func (c *Client) Stream(ctx context.Context, req provider.Request) (provider.Stream, error) {
	if responsesModel(req.ModelID) {
		resp, err := c.completeResponses(ctx, req)
		if err != nil {
			return nil, err
		}
		return &singleResponse{resp: resp}, nil
	}
	body, err := c.encodeRequest(req, true)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.authorize(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("openai chat stream: %s: %s", resp.Status, string(b))
	}
	return newSSEStream(resp.Body), nil
}

func (c *Client) authorize(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	for k, v := range c.Headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
}

func (c *Client) encodeRequest(req provider.Request, stream bool) ([]byte, error) {
	msgs := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		cm := chatMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID, Name: m.Name}
		for _, tc := range m.ToolCalls {
			cm.ToolCalls = append(cm.ToolCalls, toolCallWire{
				ID:   tc.ID,
				Type: "function",
				Function: toolCallFn{
					Name:      tc.Name,
					Arguments: string(tc.Arguments),
				},
			})
		}
		msgs = append(msgs, cm)
	}
	tools := make([]toolWire, 0, len(req.Tools))
	for _, td := range req.Tools {
		tools = append(tools, toolWire{
			Type: "function",
			Function: toolFnWire{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.Parameters,
			},
		})
	}
	wire := chatRequest{
		Model:    req.ModelID,
		Messages: msgs,
		Stream:   stream,
		Tools:    tools,
	}
	if req.ResponseFormat.Kind == provider.FormatJSONObject {
		wire.ResponseFormat = &responseFormatWire{Type: "json_object"}
	}
	if req.ResponseFormat.Kind == provider.FormatJSONSchema && req.ResponseFormat.Schema != nil {
		sb, _ := json.Marshal(req.ResponseFormat.Schema)
		wire.ResponseFormat = &responseFormatWire{
			Type: "json_schema",
			JSONSchema: &jsonSchemaWire{
				Name:   req.ResponseFormat.Name,
				Schema: sb,
			},
		}
	}
	return json.Marshal(wire)
}

func classifyHTTPError(code int, body string) error {
	low := strings.ToLower(body)
	kind := provider.APIErrorKind("")
	if code == 400 {
		if strings.Contains(low, "response_format") || strings.Contains(low, "json_schema") {
			kind = provider.APIErrorConstraintRejected
		}
	}
	if kind != "" {
		return &provider.APIError{Kind: kind, StatusCode: code, Message: body}
	}
	return fmt.Errorf("openai chat: %d: %s", code, body)
}

func decodeResponse(cr chatResponse) provider.Response {
	var out provider.Response
	if len(cr.Choices) > 0 {
		ch := cr.Choices[0]
		out.Text = ch.Message.Content
		out.FinishReason = ch.FinishReason
		for _, tc := range ch.Message.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, provider.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			})
		}
	}
	out.Usage = normalizeUsage(cr.Usage)
	return out
}

func normalizeUsage(u usageWire) provider.Usage {
	return provider.Usage{
		InputTokens:     u.PromptTokens,
		OutputTokens:    u.CompletionTokens,
		CachedTokens:    u.PromptDetails.CachedTokens,
		ReasoningTokens: u.CompletionDetails.ReasoningTokens,
		ProviderCostUSD: u.Cost,
	}
}
