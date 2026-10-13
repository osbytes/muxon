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

func TestDirectorReplayVerifyFailThenPass(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	wrongPatch, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return a - b", "replace": "return a - b"}},
	})
	goodPatch, _ := json.Marshal(map[string]interface{}{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return a - b", "replace": "return a + b"}},
	})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "done"})
	script := []provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "patch", Arguments: wrongPatch}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "2", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "3", Name: "patch", Arguments: goodPatch}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "4", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	}
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	out, err := Run(context.Background(), Config{
		Goal:        "fix test",
		Workspace:   dir,
		Trusted:     true,
		TheeVersion: "test",
		Provider:    scripted.New(script),
		Manifest:    manifest,
		TracePath:   filepath.Join(t.TempDir(), "trace.jsonl"),
		MaxRounds:   30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != StateDone || out.ExitCode != 0 {
		t.Fatalf("outcome %+v", out)
	}
}
