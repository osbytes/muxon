package director

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestRepairThenSucceeds(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	readArgs, _ := json.Marshal(map[string]interface{}{"path": "sum.go", "start_line": 1, "end_line": 10})
	patchArgs, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return a - b", "replace": "return a + b"}},
	})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "ok"})
	script := []provider.Response{
		{Text: "no native call"},
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "read", Arguments: readArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "2", Name: "patch", Arguments: patchArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "3", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	}
	manifest, _ := config.LoadEmbeddedManifest()
	out, err := Run(context.Background(), Config{
		Goal: "fix", Workspace: dir, Trusted: true, TheeVersion: "t",
		Provider: scripted.New(script), Manifest: manifest,
		TracePath: filepath.Join(t.TempDir(), "t.jsonl"), MaxRounds: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != StateDone {
		t.Fatalf("%+v", out)
	}
}
