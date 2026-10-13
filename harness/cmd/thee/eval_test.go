package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/eval"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestEvalRunWritesReward(t *testing.T) {
	home := t.TempDir()
	t.Setenv("THEE_HOME", home)
	workDir := testutil.CopyFixture(t, "go-failing-test")
	task := eval.Task{
		ID:      "fix-test-001",
		Prompt:  "fix the failing test",
		Fixture: "go-failing-test",
	}
	out, err := runEvalTask(context.Background(), task, workDir, scripted.New(director.M0ScriptedResponses()), true)
	if err != nil {
		t.Fatal(err)
	}
	if out.State != director.StateDone {
		t.Fatalf("%+v", out)
	}
	resultsPath := filepath.Join(workDir, ".thee", "eval", "results.jsonl")
	data, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"reward":1`) {
		t.Fatalf("%s", data)
	}
	var row eval.ResultRow
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &row); err != nil {
		t.Fatal(err)
	}
	if row.Reward != 1 {
		t.Fatalf("%+v", row)
	}
}
