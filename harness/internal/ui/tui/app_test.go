package tui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestViewHasRegions(t *testing.T) {
	m := NewModel(Options{Workspace: t.TempDir(), WorkerModel: "m0", Permission: "edit"})
	view := m.View()
	for _, region := range []string{"header", "transcript", "status", "input"} {
		if !strings.Contains(view, region) {
			t.Fatalf("missing %s in %s", region, view)
		}
	}
}

func TestSlashHelp(t *testing.T) {
	m := NewModel(Options{Workspace: t.TempDir()})
	m.input.SetValue("/help")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if !strings.Contains(strings.Join(model.transcript, "\n"), "/abort") {
		t.Fatalf("%v", model.transcript)
	}
}

func TestApprovalModalKeys(t *testing.T) {
	modal := NewApprovalModal()
	modal.Press("y")
	if !modal.Done || modal.Decision != "allow_once" {
		t.Fatalf("%+v", modal)
	}
}

func TestTrustModalKeys(t *testing.T) {
	modal := NewTrustModal()
	modal.Press("t")
	if modal.Decision != "trust" {
		t.Fatalf("%s", modal.Decision)
	}
}

func TestInterruptSecondCancels(t *testing.T) {
	m := NewModel(Options{Workspace: t.TempDir()})
	now := time.Now()
	m.Interrupt(now)
	m.Interrupt(now.Add(time.Second))
	if m.LastOutcome().ExitCode != 130 {
		t.Fatalf("%+v", m.LastOutcome())
	}
}

func TestScriptedFixtureDone(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(Options{
		Workspace:   dir,
		SessionID:   "tui-session",
		Trusted:     true,
		Provider:    scripted.New(director.M0ScriptedResponses()),
		Manifest:    manifest,
		Version:     "test",
		WorkerModel: "m0-worker",
		Permission:  "edit",
	})
	m.input.SetValue("fix the failing test")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected run")
	}
	msg := cmd()
	updated, _ = updated.Update(msg)
	model := updated.(Model)
	if model.LastOutcome().State != director.StateDone {
		t.Fatalf("%+v\n%s", model.LastOutcome(), model.View())
	}
	if !strings.Contains(model.View(), "verify pass") {
		t.Fatalf("view %s", model.View())
	}
	finishArgs, _ := json.Marshal(map[string]string{"summary": "still good"})
	model.opt.Provider = scripted.New([]provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "9", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	})
	model.input.SetValue("explain the sum function")
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("second goal")
	}
	updated, _ = updated.Update(cmd())
	model = updated.(Model)
	if model.LastOutcome().State != director.StateDone {
		t.Fatalf("second goal %+v", model.LastOutcome())
	}
}
