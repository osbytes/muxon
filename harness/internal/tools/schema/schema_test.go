package schema

import (
	"encoding/json"
	"testing"
)

type readArgs struct {
	Path      string `json:"path" jsonschema:"required"`
	StartLine int    `json:"start_line" jsonschema:"required"`
	EndLine   int    `json:"end_line" jsonschema:"required"`
}

func TestReadSchemaRequiresPath(t *testing.T) {
	s := FromStruct(readArgs{})
	var m map[string]interface{}
	if err := json.Unmarshal(s, &m); err != nil {
		t.Fatal(err)
	}
	req, _ := m["required"].([]interface{})
	found := false
	for _, r := range req {
		if r == "path" {
			found = true
		}
	}
	if !found {
		t.Fatalf("required: %v", req)
	}
	if err := ValidateRequired(json.RawMessage(`{"start_line":1,"end_line":2}`), s); err == nil {
		t.Fatal("expected missing path error")
	}
}
