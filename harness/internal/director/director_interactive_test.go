package director

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func runInteractive(t *testing.T, script []provider.Response, ask func(context.Context, string, []string) (string, error)) (Outcome, []string) {
	t.Helper()
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	out, err := Run(context.Background(), Config{
		Goal: "hello", Workspace: dir, Trusted: true, Interactive: true, RegisterAllTools: true, TheeVersion: "t",
		Provider: scripted.New(script), Manifest: manifest,
		TracePath:   filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds:   10,
		OnAssistant: func(text string) { said = append(said, text) },
		AskUser:     ask,
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, said
}

func TestProviderErrorIsReturned(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	out, err := Run(context.Background(), Config{
		Goal: "hello", Workspace: dir, Trusted: true, Interactive: true, TheeVersion: "t",
		Provider: scripted.New(nil), Manifest: manifest,
		TracePath: filepath.Join(t.TempDir(), "t.jsonl"),
	})
	if out.Reason != "model_error" {
		t.Fatalf("outcome %+v, err %v", out, err)
	}
	if err == nil || !strings.Contains(err.Error(), "queue exhausted") {
		t.Fatalf("Run() err = %v, want queue exhausted", err)
	}
}

func TestFollowUpKeepsPriorTurn(t *testing.T) {
	readArgs, _ := json.Marshal(map[string]any{"path": "sum.go", "start_line": 1, "end_line": 5})
	first, _ := runInteractiveGoal(t, "explain tool calling", nil, []provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "read", Arguments: readArgs}}, FinishReason: "tool_calls"},
		{Text: "Tool calls go through toolcall.Next."},
	})
	joined := historyText(first.History)
	for _, want := range []string{"explain tool calling", "sum.go", "Tool calls go through toolcall.Next."} {
		if !strings.Contains(joined, want) {
			t.Fatalf("first History = %q, want %q", joined, want)
		}
	}
	second, _ := runInteractiveGoal(t, "try again", first.History, []provider.Response{
		{Text: "Starting from the tool-calling question."},
	})
	again := historyText(second.History)
	for _, want := range []string{"explain tool calling", "sum.go", "Tool calls go through toolcall.Next.", "try again"} {
		if !strings.Contains(again, want) {
			t.Fatalf("second History = %q, want %q", again, want)
		}
	}
}

func runInteractiveGoal(t *testing.T, goal string, history []provider.Message, script []provider.Response) (Outcome, []string) {
	t.Helper()
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	out, err := Run(context.Background(), Config{
		Goal: goal, Workspace: dir, Trusted: true, Interactive: true, RegisterAllTools: true, TheeVersion: "t",
		Provider: scripted.New(script), Manifest: manifest, History: history,
		TracePath:   filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds:   10,
		OnAssistant: func(text string) { said = append(said, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, said
}

func historyText(msgs []provider.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Content)
		b.WriteByte('\n')
		for _, call := range m.ToolCalls {
			b.WriteString(call.Name)
			b.Write(call.Arguments)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func TestInteractivePlainReplyAnswers(t *testing.T) {
	out, said := runInteractive(t, []provider.Response{{Text: "Hi! What should I work on?"}}, nil)
	if out.State != StateDone || out.Reason != "answered" {
		t.Fatalf("%+v", out)
	}
	if len(said) != 1 || said[0] != "Hi! What should I work on?" {
		t.Fatalf("said %q", said)
	}
}

func TestInteractiveFinishWithoutChanges(t *testing.T) {
	finishArgs, _ := json.Marshal(map[string]string{"summary": "nothing to change"})
	out, said := runInteractive(t, []provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	}, nil)
	if out.State != StateDone || out.Reason != "no_changes" {
		t.Fatalf("%+v", out)
	}
	if len(said) != 1 || said[0] != "nothing to change" {
		t.Fatalf("said %q", said)
	}
}

func TestInteractiveAskUserWaitsForAnswer(t *testing.T) {
	askArgs, _ := json.Marshal(map[string]any{"question": "Which file?", "choices": []string{"sum.go", "main.go"}})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "ok"})
	var asked string
	var gotChoices []string
	out, _ := runInteractive(t, []provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "ask_user", Arguments: askArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "2", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	}, func(_ context.Context, q string, choices []string) (string, error) {
		asked, gotChoices = q, choices
		return "sum.go", nil
	})
	if out.State != StateDone {
		t.Fatalf("%+v", out)
	}
	if asked != "Which file?" || strings.Join(gotChoices, ",") != "sum.go,main.go" {
		t.Fatalf("asked %q choices %v", asked, gotChoices)
	}
}
