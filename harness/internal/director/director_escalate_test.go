package director

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/muxon/harness/internal/testutil"
)

func TestDirectorVerifyFailEmitsEscalate(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	wrongPatch, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return a - b", "replace": "return 0"}},
	})
	goodPatch, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return 1", "replace": "return a + b"}},
	})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "done"})
	wrongPatch2, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return 0", "replace": "return 1"}},
	})
	script := []provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "patch", Arguments: wrongPatch}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "2", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "3", Name: "patch", Arguments: wrongPatch2}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "4", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "5", Name: "patch", Arguments: goodPatch}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "6", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	}
	tracePath := filepath.Join(t.TempDir(), "trace.jsonl")
	manifest := escalateTestManifest()
	_, err := Run(context.Background(), Config{
		Goal:         "fix test",
		Workspace:    dir,
		Trusted:      true,
		MuxonVersion: "test",
		Provider:     scripted.New(script),
		Manifest:     manifest,
		TracePath:    tracePath,
		MaxRounds:    30,
		Caps: SessionCaps{
			MaxToolRounds:            30,
			MaxVerifyAttempts:        6,
			EscalateAfterVerifyFails: 2,
			MaxEscalations:           1,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "queue exhausted") {
		t.Fatal(err)
	}
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"type":"escalate"`) {
		t.Fatalf("missing escalate event: %s", data)
	}
}

func escalateTestManifest() *config.Manifest {
	m, err := config.LoadEmbeddedManifest()
	if err != nil {
		panic(err)
	}
	m.Roles["escalated"] = config.RoleDef{ModelID: "m0-escalated"}
	m.Models = append(m.Models, config.ModelDef{
		ID:       "m0-escalated",
		Provider: "deepinfra",
		Model:    "test/escalated",
		Capabilities: config.Capabilities{
			ContextWindow: 32768,
			ToolProtocol:  "native",
			EditFormat:    "search_replace",
			PromptVariant: "compact",
		},
	})
	return m
}
