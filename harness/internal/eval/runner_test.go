package eval

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/director"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider/scripted"
)

func TestHiddenCheckOverridesDirectorDone(t *testing.T) {
	base := t.TempDir()
	checkDir := filepath.Join(base, "checks", "hidden")
	if err := os.MkdirAll(checkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("package sum\n\nimport \"testing\"\n\nfunc TestHidden(t *testing.T) {\n\tif Sum(2, 3) != 99 {\n\t\tt.Fatal(Sum(2, 3))\n\t}\n}\n")
	if err := os.WriteFile(filepath.Join(checkDir, "hidden_test.go"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	task := Task{
		ID:      "fix-test-001",
		Prompt:  "fix the failing test",
		Fixture: "go-failing-test",
		Check:   Check{Cmd: []string{"go", "test", "."}, Files: "checks/hidden"},
	}
	res, err := RunTrial(context.Background(), task, TrialOptions{
		Provider: scripted.New(director.M0ScriptedResponses()),
		Trusted:  true,
		TaskDir:  base,
		Trial:    1,
		Timeout:  time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome.State != director.StateDone {
		t.Fatalf("director state %s reason %s", res.Outcome.State, res.Outcome.Reason)
	}
	if res.Row.Reward != 0 {
		t.Fatalf("reward %d, want 0 from hidden check", res.Row.Reward)
	}
}

func TestTrialsUseSeparateWorkspaces(t *testing.T) {
	task := Task{ID: "fix-test-001", Prompt: "fix the failing test", Fixture: "go-failing-test"}
	opt := TrialOptions{Trusted: true, KeepWorkspace: true, Timeout: time.Minute}
	opt.Provider = scripted.New(director.M0ScriptedResponses())
	first, err := RunTrial(context.Background(), task, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(first.Workspace) })
	opt.Provider = scripted.New(director.M0ScriptedResponses())
	second, err := RunTrial(context.Background(), task, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(second.Workspace) })
	if first.Workspace == "" || first.Workspace == second.Workspace {
		t.Fatalf("workspaces %q and %q", first.Workspace, second.Workspace)
	}
	marker := filepath.Join(first.Workspace, "marker-from-trial-1")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(second.Workspace, "marker-from-trial-1")); !os.IsNotExist(err) {
		t.Fatalf("second workspace saw first trial marker: %v", err)
	}
}

type blockProvider struct{}

func (blockProvider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	<-ctx.Done()
	return provider.Response{}, ctx.Err()
}

func (blockProvider) Stream(ctx context.Context, req provider.Request) (provider.Stream, error) {
	_, err := blockProvider{}.Complete(ctx, req)
	return nil, err
}

func TestTimeoutIsFailure(t *testing.T) {
	task := Task{ID: "fix-test-001", Prompt: "fix the failing test", Fixture: "go-failing-test"}
	res, err := RunTrial(context.Background(), task, TrialOptions{
		Provider: blockProvider{},
		Trusted:  true,
		Timeout:  200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Row.TimeoutHit || res.Row.Reward != 0 || res.Row.Reason != "timeout" {
		t.Fatalf("%+v", res.Row)
	}
}

func TestBasketStopsAtBudget(t *testing.T) {
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	md, err := manifest.ResolveModel("m0-worker")
	if err != nil {
		t.Fatal(err)
	}
	md.InputPerM = 100
	md.OutputPerM = 100
	task := PreparedTask{Task: Task{
		ID: "fix-test-001", Prompt: "fix the failing test", Fixture: "go-failing-test",
	}}
	script := func(string) (provider.Provider, error) {
		rs := director.M0ScriptedResponses()
		for i := range rs {
			rs[i].Usage.InputTokens = 100000
			rs[i].Usage.OutputTokens = 1000
		}
		return scripted.New(rs), nil
	}
	out := t.TempDir()
	header, rows, err := RunBasket(context.Background(), []PreparedTask{task, task}, []ModelSpec{{ModelID: "m0-worker"}}, BasketOptions{
		Trials:      1,
		Concurrency: 1,
		BudgetUSD:   5,
		OutputDir:   out,
		Trusted:     true,
		Manifest:    manifest,
		ProviderFor: script,
		RunID:       "budget-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if header.StoppedReason != "budget" {
		t.Fatalf("stopped %q header %+v rows %d", header.StoppedReason, header, len(rows))
	}
	if len(rows) != 1 {
		t.Fatalf("rows %d, want 1", len(rows))
	}
	if rows[0].CostUSD < 5 {
		t.Fatalf("cost %v, want at least the budget", rows[0].CostUSD)
	}
}
