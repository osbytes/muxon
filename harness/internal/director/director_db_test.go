package director

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/db"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/muxon/harness/internal/testutil"

	_ "modernc.org/sqlite"
)

func TestRejectedPatchIsStored(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	patchArgs, _ := json.Marshal(map[string]any{
		"path":   "sum.go",
		"format": "search_replace",
		"replacements": []map[string]string{{
			"search":  "this text is not in the file",
			"replace": "nope",
		}},
	})
	out, err := Run(context.Background(), Config{
		Goal: "fix the sum", Workspace: dir, Trusted: true, Interactive: true, MuxonVersion: "t",
		Provider: scripted.New([]provider.Response{
			{
				ToolCalls:    []provider.ToolCall{{ID: "call_patch", Name: "patch", Arguments: patchArgs}},
				FinishReason: "tool_calls",
				Usage:        provider.Usage{InputTokens: 11, OutputTokens: 5},
			},
			{Text: "The patch missed."},
		}),
		Manifest:  manifest,
		TracePath: filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != StateDone || out.Reason != "answered" {
		t.Fatalf("Run() outcome = %+v, want done answered", out)
	}

	sqlDB, err := sql.Open("sqlite", db.File(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	var ok int
	var result string
	if err := sqlDB.QueryRow(`SELECT ok, result FROM tool_calls WHERE name = ?`, "patch").Scan(&ok, &result); err != nil {
		t.Fatalf("tool_calls query: %v", err)
	}
	if ok != 0 || !strings.Contains(result, "patch rejected") {
		t.Fatalf("tool_calls ok = %d, result = %q, want a patch rejection", ok, result)
	}

	var level, message, source, logErr string
	if err := sqlDB.QueryRow(`SELECT level, message, source, error FROM logs WHERE source = ?`, "patch").Scan(&level, &message, &source, &logErr); err != nil {
		t.Fatalf("logs query: %v", err)
	}
	if level != "error" || message != "tool call failed" || source != "patch" || !strings.Contains(logErr, "patch rejected") {
		t.Fatalf("logs = level %q message %q source %q error %q, want the patch rejection", level, message, source, logErr)
	}

	var request string
	var inputTokens int
	if err := sqlDB.QueryRow(`SELECT request_json, input_tokens FROM turns WHERE input_tokens = ?`, 11).Scan(&request, &inputTokens); err != nil {
		t.Fatalf("turns query: %v", err)
	}
	if request == "" || inputTokens != 11 {
		t.Fatalf("turns request = %q, input_tokens = %d", request, inputTokens)
	}
}

func TestToolParseFailureIsLogged(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	out, err := Run(context.Background(), Config{
		Goal: "fix the sum", Workspace: dir, Trusted: true, MuxonVersion: "t",
		Provider: scripted.New([]provider.Response{
			{Text: "I will describe the fix instead of calling a tool."},
			{Text: "still no tool call"},
		}),
		Manifest:  manifest,
		TracePath: filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds: 10,
	})
	if out.Reason != "tool_parse_exhausted" {
		t.Fatalf("Run() outcome = %+v, err = %v, want tool_parse_exhausted", out, err)
	}

	sqlDB, err := sql.Open("sqlite", db.File(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query(`SELECT level, message, source, error FROM logs ORDER BY seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var level, message, source, logErr string
		if err := rows.Scan(&level, &message, &source, &logErr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(logErr, "no tool calls in response") || !strings.Contains(logErr, "model text:") {
			t.Fatalf("logs error = %q, want the parse error and model text", logErr)
		}
		got = append(got, level+" "+message+" "+source)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"warn tool call parse failed toolcall",
		"error tool_parse_exhausted session",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("logs = %v, want %v", got, want)
	}
}

func TestToolRoundsExhaustedIsLogged(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	readArgs, _ := json.Marshal(map[string]any{"path": "sum.go", "start_line": 1, "end_line": 5})
	read := provider.Response{
		ToolCalls:    []provider.ToolCall{{ID: "1", Name: "read", Arguments: readArgs}},
		FinishReason: "tool_calls",
	}
	out, err := Run(context.Background(), Config{
		Goal: "explain sum.go", Workspace: dir, Trusted: true, MuxonVersion: "t",
		Provider:  scripted.New([]provider.Response{read, read, read}),
		Manifest:  manifest,
		TracePath: filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds: 3,
	})
	if err != nil || out.Reason != "tool_rounds_exhausted" {
		t.Fatalf("Run() outcome = %+v, err = %v, want tool_rounds_exhausted", out, err)
	}

	level, message, source, logErr := sessionLog(t, dir)
	if level != "error" || message != "tool_rounds_exhausted" || source != "session" || !strings.Contains(logErr, "max 3") {
		t.Fatalf("logs = level %q message %q source %q error %q", level, message, source, logErr)
	}
}

func TestVerifyFailureIsLogged(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	patchArgs, _ := json.Marshal(map[string]any{
		"path": "sum.go", "format": "search_replace",
		"replacements": []map[string]string{{"search": "return a - b", "replace": "return a + b"}},
	})
	patch := provider.Response{
		ToolCalls:    []provider.ToolCall{{ID: "p", Name: "patch", Arguments: patchArgs}},
		FinishReason: "tool_calls",
	}
	finishArgs, _ := json.Marshal(map[string]string{"summary": "done"})
	finish := provider.Response{
		ToolCalls:    []provider.ToolCall{{ID: "1", Name: "finish", Arguments: finishArgs}},
		FinishReason: "tool_calls",
	}
	out, err := Run(context.Background(), Config{
		Goal: "fix the sum", Workspace: dir, Trusted: true, MuxonVersion: "t",
		Provider:  scripted.New([]provider.Response{patch, finish, finish}),
		Manifest:  manifest,
		TracePath: filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds: 10,
		VerifyCmd: "false",
		Caps: SessionCaps{
			MaxToolRounds:     10,
			MaxVerifyAttempts: 1,
			TokenBudget:       2_000_000,
		},
	})
	if err != nil || out.Reason != "verify_exhausted" {
		t.Fatalf("Run() outcome = %+v, err = %v, want verify_exhausted", out, err)
	}

	rows := chatLogs(t, dir)
	if len(rows) != 2 {
		t.Fatalf("logs = %+v, want verify failed then verify_exhausted", rows)
	}
	if rows[0].level != "warn" || rows[0].message != "verify failed" || rows[0].source != "verify" || !strings.Contains(rows[0].errText, "exit 1") {
		t.Fatalf("verify log = %+v", rows[0])
	}
	if rows[1].level != "error" || rows[1].message != "verify_exhausted" || rows[1].source != "session" || !strings.Contains(rows[1].errText, "max 1") {
		t.Fatalf("session log = %+v", rows[1])
	}
}

type chatLog struct {
	level, message, source, errText string
}

func sessionLog(t *testing.T, workspace string) (level, message, source, errText string) {
	t.Helper()
	rows := chatLogs(t, workspace)
	for _, row := range rows {
		if row.source == "session" {
			return row.level, row.message, row.source, row.errText
		}
	}
	t.Fatalf("no session log in %+v", rows)
	return "", "", "", ""
}

func chatLogs(t *testing.T, workspace string) []chatLog {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", db.File(workspace))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query(`SELECT level, message, source, error FROM logs ORDER BY seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got []chatLog
	for rows.Next() {
		var row chatLog
		if err := rows.Scan(&row.level, &row.message, &row.source, &row.errText); err != nil {
			t.Fatal(err)
		}
		got = append(got, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return got
}
