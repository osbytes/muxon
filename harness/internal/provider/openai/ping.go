package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Ping sends a one-message chat request to confirm the key and model work.
func (c *Client) Ping(ctx context.Context, model string) error {
	body, err := json.Marshal(map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": "Reply with OK."}},
		"max_tokens": 16,
		"stream":     false,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("reach %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("%s: %s", resp.Status, errorMessage(b))
}

func errorMessage(body []byte) string {
	var wire struct {
		Error any `json:"error"`
	}
	if err := json.Unmarshal(body, &wire); err == nil && wire.Error != nil {
		switch e := wire.Error.(type) {
		case string:
			return e
		case map[string]any:
			if msg, ok := e["message"].(string); ok && msg != "" {
				return msg
			}
		}
	}
	msg := strings.TrimSpace(string(body))
	if len(msg) > 200 {
		msg = msg[:200] + "…"
	}
	if msg == "" {
		return "no response body"
	}
	return msg
}
