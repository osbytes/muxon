package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

type modelEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type modelListResponse struct {
	Data   []modelEntry `json:"data"`
	Models []modelEntry `json:"models"`
}

// ListModels returns model ids from the provider's OpenAI-compatible /models endpoint.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list models: %s: %s", resp.Status, errorMessage(body))
	}
	var wire modelListResponse
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	entries := wire.Data
	if len(entries) == 0 {
		entries = wire.Models
	}
	seen := map[string]bool{}
	var names []string
	for _, entry := range entries {
		id := entry.ID
		if id == "" {
			id = entry.Name
		}
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		names = append(names, id)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("list models: empty")
	}
	sort.Strings(names)
	return names, nil
}
