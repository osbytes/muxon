package director

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/db"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"

	_ "modernc.org/sqlite"
)

func TestRejectedPatchIsStored(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
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
		Goal: "fix the sum", Workspace: dir, Trusted: true, Interactive: true, TheeVersion: "t",
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

	var request string
	var inputTokens int
	if err := sqlDB.QueryRow(`SELECT request_json, input_tokens FROM turns WHERE input_tokens = ?`, 11).Scan(&request, &inputTokens); err != nil {
		t.Fatalf("turns query: %v", err)
	}
	if request == "" || inputTokens != 11 {
		t.Fatalf("turns request = %q, input_tokens = %d", request, inputTokens)
	}
}
