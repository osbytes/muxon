package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
)

// ProbeSnippet is a suggested capability block. Thresholds are still open in the model-router doc.
type ProbeSnippet struct {
	ModelID          string `json:"model_id"`
	StructuredOutput string `json:"structured_output"`
	ToolProtocol     string `json:"tool_protocol"`
	Note             string `json:"note"`
}

// ProbeChat asks an OpenAI-compatible endpoint which structured output and tool protocol it accepted.
func ProbeChat(ctx context.Context, baseURL, apiKey, modelID string) (ProbeSnippet, error) {
	out := ProbeSnippet{
		ModelID:          modelID,
		StructuredOutput: "none",
		ToolProtocol:     "two_step",
		Note:             "thresholds open in model-router doc; recorded the setting this endpoint accepted",
	}
	schemaBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{{
			"role": "user", "content": "Reply with JSON {\"ok\":true}",
		}},
		"response_format": map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "probe",
				"schema": map[string]string{"type": "object"},
			},
		},
	}
	if _, err := postChat(ctx, baseURL, apiKey, schemaBody); err == nil {
		out.StructuredOutput = "json_schema"
	}
	toolBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{{
			"role": "user", "content": "call the ping tool",
		}},
		"tools": []map[string]interface{}{{
			"type": "function",
			"function": map[string]interface{}{
				"name":       "ping",
				"parameters": map[string]string{"type": "object"},
			},
		}},
	}
	raw, err := postChat(ctx, baseURL, apiKey, toolBody)
	if err == nil && bytes.Contains(raw, []byte(`"tool_calls"`)) {
		out.ToolProtocol = "native"
	}
	return out, nil
}

func postChat(ctx context.Context, baseURL, apiKey string, body interface{}) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return raw, fmt.Errorf("probe: %s", resp.Status)
	}
	return raw, nil
}

// WriteSnippet merges a probe result into the user catalog without storing secrets.
func WriteSnippet(snippet ProbeSnippet) error {
	path, err := config.CatalogPath()
	if err != nil {
		return err
	}
	cat, err := config.LoadCatalog()
	if err != nil {
		return err
	}
	found := false
	for i := range cat.Models {
		if cat.Models[i].ID == snippet.ModelID {
			cat.Models[i].Capabilities.StructuredOut = snippet.StructuredOutput
			cat.Models[i].Capabilities.ToolProtocol = snippet.ToolProtocol
			found = true
			break
		}
	}
	if !found {
		md := config.ModelDef{ID: snippet.ModelID, Model: snippet.ModelID}
		md.Capabilities.StructuredOut = snippet.StructuredOutput
		md.Capabilities.ToolProtocol = snippet.ToolProtocol
		cat.Models = append(cat.Models, md)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
