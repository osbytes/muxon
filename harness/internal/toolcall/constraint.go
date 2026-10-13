package toolcall

import (
	"encoding/json"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// Step1Schema returns JSON schema for tool name selection.
func Step1Schema(toolNames []string) provider.ResponseFormat {
	enum := make([]interface{}, len(toolNames))
	for i, n := range toolNames {
		enum[i] = n
	}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tool": map[string]interface{}{
				"type": "string",
				"enum": enum,
			},
		},
		"required":             []string{"tool"},
		"additionalProperties": false,
	}
	return provider.ResponseFormat{
		Kind:   provider.FormatJSONSchema,
		Schema: schema,
		Name:   "tool_pick",
	}
}

// Step2Schema wraps a tool parameter schema.
func Step2Schema(params json.RawMessage) provider.ResponseFormat {
	var m map[string]interface{}
	if err := json.Unmarshal(params, &m); err != nil {
		m = map[string]interface{}{"type": "object"}
	}
	return provider.ResponseFormat{
		Kind:   provider.FormatJSONSchema,
		Schema: m,
		Name:   "tool_args",
	}
}

// DowngradeFormat moves one step down the structured output ladder.
func DowngradeFormat(f provider.ResponseFormat) provider.ResponseFormat {
	if f.Kind == provider.FormatJSONSchema {
		return provider.ResponseFormat{Kind: provider.FormatJSONObject}
	}
	return provider.ResponseFormat{Kind: provider.FormatNone}
}
