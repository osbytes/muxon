package director

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
	"github.com/PLACEHOLDER/thee/harness/internal/verifier"
)

func TestDirectorScriptedFixture(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(t.TempDir(), "trace.jsonl")
	out, err := Run(context.Background(), Config{
		Goal:        "fix the failing test",
		Workspace:   dir,
		Trusted:     true,
		TheeVersion: "test",
		Provider:    scripted.New(M0ScriptedResponses()),
		Manifest:    manifest,
		TracePath:   trace,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ExitCode != 0 || out.State != "done" {
		t.Fatalf("outcome: %+v", out)
	}
	cmd, ok := verifier.ResolveCommand(dir)
	if !ok {
		t.Fatal("no verify cmd")
	}
	vres, err := verifier.Run(context.Background(), dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !vres.Passed {
		t.Fatalf("tests should pass: %s", vres.Excerpt)
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(data), "verify.run", "session.end", `"outcome":"done"`) {
		t.Fatalf("trace missing events: %s", data)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
