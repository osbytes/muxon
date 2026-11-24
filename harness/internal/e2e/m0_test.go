package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/director"
	"github.com/PLACEHOLDER/muxon/harness/internal/testutil"
)

func TestM0RunE2E(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MUXON_HOME", home)
	fixtureDir := testutil.CopyFixture(t, "go-failing-test")
	scriptPath := filepath.Join(home, "script.json")
	b, err := json.Marshal(director.M0ScriptedResponses())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := buildMuxon(t)
	cmd := exec.Command(bin, "run", "-p", "fix the failing test", "--trust", "--directory", fixtureDir)
	cmd.Env = append(os.Environ(),
		"MUXON_HOME="+home,
		"MUXON_SCRIPTED_PROVIDER_FILE="+scriptPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	tracePath, err := latestTrace(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "verify.run") || !strings.Contains(s, "session.end") || !strings.Contains(s, `"outcome":"done"`) {
		t.Fatalf("trace: %s", s)
	}
}

func latestTrace(fixtureDir string) (string, error) {
	sessionsDir := filepath.Join(fixtureDir, ".muxon", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "", err
	}
	var latest string
	for _, e := range entries {
		if e.IsDir() {
			latest = e.Name()
		}
	}
	if latest == "" {
		return "", os.ErrNotExist
	}
	return filepath.Join(sessionsDir, latest, "trace.jsonl"), nil
}

func buildMuxon(t *testing.T) string {
	dir := t.TempDir()
	name := "muxon"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(dir, name)
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = filepath.Join("..", "..", "cmd", "muxon")
	if outCombined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build muxon: %v\n%s", err, outCombined)
	}
	return out
}
