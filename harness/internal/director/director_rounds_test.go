package director

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/muxon/harness/internal/testutil"
)

func TestResolveCapsInteractiveDefault(t *testing.T) {
	interactive := ResolveCaps(true, SessionCaps{}, 0, config.UserConfig{})
	if interactive.MaxToolRounds != DefaultInteractiveMaxToolRounds {
		t.Fatalf("interactive MaxToolRounds = %d, want %d", interactive.MaxToolRounds, DefaultInteractiveMaxToolRounds)
	}
	if interactive.MaxVerifyAttempts != DefaultCaps().MaxVerifyAttempts {
		t.Fatalf("interactive lost other defaults: %+v", interactive)
	}
	headless := ResolveCaps(false, SessionCaps{}, 0, config.UserConfig{})
	if headless.MaxToolRounds != DefaultCaps().MaxToolRounds {
		t.Fatalf("headless MaxToolRounds = %d, want %d", headless.MaxToolRounds, DefaultCaps().MaxToolRounds)
	}
	flagged := ResolveCaps(true, SessionCaps{}, 12, config.UserConfig{})
	if flagged.MaxToolRounds != 12 {
		t.Fatalf("flag MaxToolRounds = %d, want 12", flagged.MaxToolRounds)
	}
	var uc config.UserConfig
	uc.Session.MaxToolRounds = 25
	configured := ResolveCaps(true, SessionCaps{}, 0, uc)
	if configured.MaxToolRounds != 25 {
		t.Fatalf("config MaxToolRounds = %d, want 25", configured.MaxToolRounds)
	}
	explicit := ResolveCaps(true, SessionCaps{MaxToolRounds: 7}, 0, config.UserConfig{})
	if explicit.MaxToolRounds != 7 {
		t.Fatalf("explicit MaxToolRounds = %d, want 7", explicit.MaxToolRounds)
	}
}

func TestInteractivePausesAtToolRoundLimit(t *testing.T) {
	summary := "Stopped after reading sum.go. Next: patch the return."
	out, said := runRoundLimit(t, true, summary)
	if out.State != StatePaused || out.Reason != "tool_rounds_exhausted" || out.ExitCode != 0 {
		t.Fatalf("outcome %+v", out)
	}
	joined := historyText(out.History)
	if !strings.Contains(joined, summary) || !strings.Contains(joined, toolRoundLimitSummary) {
		t.Fatalf("history missing summary: %s", joined)
	}
	if len(said) != 1 || said[0] != summary {
		t.Fatalf("OnAssistant = %q", said)
	}
}

func TestHeadlessFailsAtToolRoundLimit(t *testing.T) {
	summary := "this summary must not be requested"
	out, said := runRoundLimit(t, false, summary)
	if out.State != StateFailed || out.Reason != "tool_rounds_exhausted" || out.ExitCode != 1 {
		t.Fatalf("outcome %+v", out)
	}
	if strings.Contains(historyText(out.History), summary) || strings.Contains(historyText(out.History), toolRoundLimitSummary) {
		t.Fatalf("headless history included the summary: %s", historyText(out.History))
	}
	if len(said) != 0 {
		t.Fatalf("OnAssistant = %q", said)
	}
}

func runRoundLimit(t *testing.T, interactive bool, summary string) (Outcome, []string) {
	t.Helper()
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	readArgs, _ := json.Marshal(map[string]any{"path": "sum.go", "start_line": 1, "end_line": 5})
	read := func(id string) provider.Response {
		return provider.Response{
			ToolCalls:    []provider.ToolCall{{ID: id, Name: "read", Arguments: readArgs}},
			FinishReason: "tool_calls",
		}
	}
	var said []string
	out, err := Run(context.Background(), Config{
		Goal: "explain sum.go", Workspace: dir, Trusted: true, Interactive: interactive,
		RegisterAllTools: true, MuxonVersion: "t",
		Provider: scripted.New([]provider.Response{
			read("1"), read("2"), read("3"),
			{Text: summary},
		}),
		Manifest:    manifest,
		TracePath:   filepath.Join(t.TempDir(), "t.jsonl"),
		MaxRounds:   3,
		OnAssistant: func(text string) { said = append(said, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return out, said
}
