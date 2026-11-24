package toolcall

import (
	"encoding/json"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

func TestSalvageFencedTrailingComma(t *testing.T) {
	spec := tools.Spec{
		Name:       "read",
		Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
	}
	text := "```json\n{\"path\":\"a.go\",}\n```"
	res, err := SalvageText(text, "read", spec)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Salvaged {
		t.Fatal("expected salvaged")
	}
	var m map[string]string
	if err := json.Unmarshal(res.Call.Arguments, &m); err != nil || m["path"] != "a.go" {
		t.Fatalf("%+v", res.Call.Arguments)
	}
}
