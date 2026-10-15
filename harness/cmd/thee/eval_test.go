package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/eval"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
)

func TestEvalRunWritesReward(t *testing.T) {
	home := t.TempDir()
	t.Setenv("THEE_HOME", home)
	results := filepath.Join(t.TempDir(), "results.jsonl")
	task := eval.Task{
		ID:      "fix-test-001",
		Prompt:  "fix the failing test",
		Fixture: "go-failing-test",
	}
	res, err := eval.RunTrial(context.Background(), task, eval.TrialOptions{
		Provider:    scripted.New(director.M0ScriptedResponses()),
		Trusted:     true,
		ResultsPath: results,
		Timeout:     0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome.State != director.StateDone {
		t.Fatalf("%+v", res.Outcome)
	}
	if res.Row.Reward != 1 {
		t.Fatalf("%+v", res.Row)
	}
	data, err := os.ReadFile(results)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"reward":1`) {
		t.Fatalf("%s", data)
	}
	rows, err := eval.LoadResults(results)
	if err != nil {
		t.Fatal(err)
	}
	report := eval.FormatReport(rows)
	if !strings.Contains(report, "pass rate        1.000") {
		t.Fatalf("%s", report)
	}
}
