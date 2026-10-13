package director

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestDirectorQuestionGoalClassification(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	tracePath := filepath.Join(t.TempDir(), "trace.jsonl")
	manifest, _ := config.LoadEmbeddedManifest()
	_, err := Run(context.Background(), Config{
		Goal:        "where is sum configured?",
		Workspace:   dir,
		Trusted:     true,
		TheeVersion: "test",
		Provider:    scripted.New(M0ScriptedResponses()),
		Manifest:    manifest,
		TracePath:   tracePath,
		MaxRounds:   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(tracePath)
	if !strings.Contains(string(data), `"task_type":"question"`) {
		t.Fatalf("trace %s", data)
	}
}
