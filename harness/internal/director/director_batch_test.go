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

func manifestWithParallel(t *testing.T, on bool) *config.Manifest {
	t.Helper()
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	worker, err := manifest.ResolveRole("worker")
	if err != nil {
		t.Fatal(err)
	}
	worker.Capabilities.ParallelToolCalls = on
	worker.Capabilities.ToolProtocol = "native"
	return manifest
}

func readCall(id, path string) provider.ToolCall {
	args, _ := json.Marshal(map[string]any{"path": path, "start_line": 1, "end_line": 2})
	return provider.ToolCall{ID: id, Name: "read", Arguments: args}
}

func runBatch(t *testing.T, parallel bool, allTools bool, script []provider.Response) (Outcome, string) {
	t.Helper()
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	trace := filepath.Join(t.TempDir(), "t.jsonl")
	out, err := Run(context.Background(), Config{
		Goal: "look around", Workspace: dir, Trusted: true, Interactive: true,
		RegisterAllTools: allTools, MuxonVersion: "t",
		Provider:  scripted.New(script),
		Manifest:  manifestWithParallel(t, parallel),
		TracePath: trace,
		MaxRounds: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, trace
}

func TestBatchOfReadsIsOneRound(t *testing.T) {
	out, trace := runBatch(t, true, false, []provider.Response{
		{ToolCalls: []provider.ToolCall{
			readCall("1", "sum.go"),
			readCall("2", "sum_test.go"),
			readCall("3", "go.mod"),
		}, FinishReason: "tool_calls"},
		{Text: "Three files, one turn."},
	})
	if out.State != StateDone || out.Reason != "answered" {
		t.Fatalf("outcome %+v", out)
	}
	joined := historyText(out.History)
	for _, want := range []string{"sum.go", "sum_test.go", "go.mod"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("history missing %s: %s", want, joined)
		}
	}
	if strings.Contains(joined, "Emit one valid tool call") || strings.Contains(joined, "Emit at most") {
		t.Fatalf("batch took a repair turn: %s", joined)
	}
	if rounds := toolRoundsInTrace(t, trace); rounds != 1 {
		t.Fatalf("tool_rounds = %d, want 1", rounds)
	}
}

func TestFinishEndsBatch(t *testing.T) {
	finishArgs, _ := json.Marshal(map[string]string{"summary": "nothing to change"})
	out, _ := runBatch(t, true, false, []provider.Response{{
		ToolCalls: []provider.ToolCall{
			{ID: "1", Name: "finish", Arguments: finishArgs},
			readCall("2", "sum.go"),
		},
		FinishReason: "tool_calls",
	}})
	if out.State != StateDone {
		t.Fatalf("outcome %+v", out)
	}
	joined := historyText(out.History)
	if !strings.Contains(joined, "not run: finish was called earlier in this batch") {
		t.Fatalf("later call was run: %s", joined)
	}
}

func TestFailedFinishDoesNotEndGoal(t *testing.T) {
	todoArgs, _ := json.Marshal(map[string]string{"action": "add", "text": "left"})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "done"})
	out, _ := runBatch(t, true, true, []provider.Response{
		{ToolCalls: []provider.ToolCall{
			{ID: "1", Name: "todo", Arguments: todoArgs},
			{ID: "2", Name: "finish", Arguments: finishArgs},
		}, FinishReason: "tool_calls"},
		{Text: "Todos are still open."},
	})
	if out.Reason != "answered" {
		t.Fatalf("outcome %+v", out)
	}
	joined := historyText(out.History)
	if !strings.Contains(joined, "open todo items remain") {
		t.Fatalf("finish failure missing: %s", joined)
	}
}

func TestMultipleCallsRepairWhenParallelOff(t *testing.T) {
	out, _ := runBatch(t, false, false, []provider.Response{
		{ToolCalls: []provider.ToolCall{
			readCall("1", "sum.go"),
			readCall("2", "sum_test.go"),
		}, FinishReason: "tool_calls"},
		{Text: "One at a time."},
	})
	if out.Reason != "answered" {
		t.Fatalf("outcome %+v", out)
	}
	joined := historyText(out.History)
	if !strings.Contains(joined, "Emit one valid tool call.") {
		t.Fatalf("repair message missing: %s", joined)
	}
}

func toolRoundsInTrace(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, `"session.end"`) {
			continue
		}
		var ev struct {
			ToolRounds int `json:"tool_rounds"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatal(err)
		}
		return ev.ToolRounds
	}
	t.Fatal("session.end missing")
	return 0
}
