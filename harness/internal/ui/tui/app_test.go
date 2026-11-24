package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/director"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/muxon/harness/internal/security"
	"github.com/PLACEHOLDER/muxon/harness/internal/testutil"
)

func sessionModel(dir string) Model {
	m := NewModel(Options{
		Workspace:   dir,
		WorkerModel: "m0",
		Permission:  "edit",
		Provider:    scripted.New(nil),
		Trusted:     true,
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return updated.(Model)
}

func TestViewHasRegions(t *testing.T) {
	dir := t.TempDir()
	m := sessionModel(dir)
	view := m.View()
	for _, want := range []string{filepath.Base(dir), "/providers", "enter send", "╭", "m0"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "header ") {
		t.Errorf("View() still has the debug header:\n%s", view)
	}
}

func TestTranscriptRendersBlocks(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.appendLine("> fix the test")
	runTool(&m, "patch", `{"path":"sum.go"}`, true, "")
	m.appendLine("▸ verify pass")
	m.appendLine("[done] verified")
	view := m.View()
	for _, want := range []string{"fix the test", "Edit sum.go", "⎿  applied", "Verify", "⎿  passed", "● done · verified"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q:\n%s", want, view)
		}
	}
}

func runTool(m *Model, name, args string, ok bool, output string) {
	m.applyTool(director.ToolActivity{Name: name, Args: json.RawMessage(args)})
	m.applyTool(director.ToolActivity{Name: name, Args: json.RawMessage(args), Done: true, OK: ok, Output: output})
}

func TestToolRowsShowArgsAndSummary(t *testing.T) {
	m := sessionModel(t.TempDir())
	runTool(&m, "search", `{"pattern":"agent loop"}`, true, "a.go:1:x\nb.go:2:y\n")
	m.appendLine("[assistant] reading")
	runTool(&m, "read", `{"path":"internal/director/director.go","start_line":1,"end_line":3}`, true, "a\nb\nc")
	view := m.View()
	for _, want := range []string{`Search "agent loop"`, "2 matches", "Read internal/director/director.go:1-3", "3 lines"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestConsecutiveReadsCollapse(t *testing.T) {
	m := sessionModel(t.TempDir())
	for _, p := range []string{"a.go", "b.go", "c.go", "d.go", "e.go"} {
		runTool(&m, "read", `{"path":"`+p+`","start_line":1,"end_line":2}`, true, "x\ny")
	}
	runTool(&m, "read", `{"path":"missing.go","start_line":1,"end_line":2}`, false, "file not found")
	view := m.View()
	for _, want := range []string{"Read 6 files", "1 failed", "+3 more", "ctrl+o expand"} {
		if !strings.Contains(view, want) {
			t.Errorf("collapsed View() missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Read b.go") {
		t.Errorf("collapsed View() lists individual reads:\n%s", view)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	view = updated.(Model).View()
	for _, want := range []string{"Read e.go:1-2", "Read missing.go:1-2", "file not found"} {
		if !strings.Contains(view, want) {
			t.Errorf("expanded View() missing %q:\n%s", want, view)
		}
	}
}

func TestHiddenToolsSkipTranscript(t *testing.T) {
	m := sessionModel(t.TempDir())
	runTool(&m, "ask_user", `{"question":"q"}`, true, "")
	runTool(&m, "finish", `{"summary":"s"}`, true, "")
	if len(m.transcript) != 0 {
		t.Fatalf("transcript = %q, want empty", m.transcriptText())
	}
}

func TestMainScreenWithoutProvider(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(Options{Workspace: dir, WorkerModel: "m0", Permission: "edit", Trusted: true})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	view := updated.(Model).View()
	if strings.Contains(view, "Connect a provider") || strings.Contains(view, "DEEPINFRA") {
		t.Fatalf("startup asked for a provider account: %s", view)
	}
	if !strings.Contains(view, filepath.Base(dir)) || !strings.Contains(view, "/providers") {
		t.Fatalf("main screen missing workspace or /providers: %s", view)
	}
}

type fakeOpenCode struct {
	status  int
	body    string
	auth    string
	session string
	model   string
}

func setupOpenCode(t *testing.T, fake *fakeOpenCode, secret string) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("MUXON_HOME", home)
	t.Setenv("OPENCODE_API_KEY", "")
	t.Setenv("DEEPINFRA_API_KEY", "")
	auth := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(auth, []byte(`{"opencode-go":{"type":"api","key":"`+secret+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MUXON_OPENCODE_AUTH", auth)
	config.SetSecretStore(config.NewMemoryKeyring())
	t.Cleanup(func() { config.SetSecretStore(nil) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		fake.auth = r.Header.Get("Authorization")
		fake.session = r.Header.Get("x-opencode-session")
		fake.model = req.Model
		w.WriteHeader(fake.status)
		_, _ = w.Write([]byte(fake.body))
	}))
	t.Cleanup(srv.Close)
	old := openCodeGoBaseURL
	openCodeGoBaseURL = srv.URL
	t.Cleanup(func() { openCodeGoBaseURL = old })
	return home
}

func pickOpenCode(t *testing.T, m Model, open string) Model {
	t.Helper()
	m.input.SetValue(open)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.pickerOpen {
		t.Fatalf("%s did not open the picker", open)
	}
	view := m.View()
	if !strings.Contains(view, "› OpenCode Go") || !strings.Contains(view, "key found") {
		t.Fatalf("picker does not highlight OpenCode Go with a found key:\n%s", view)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil || m.connecting != "OpenCode Go" {
		t.Fatalf("selecting OpenCode Go did not start a connection (connecting=%q)", m.connecting)
	}
	updated, _ = m.Update(cmd())
	return updated.(Model)
}

func TestProvidersConnectsOpenCode(t *testing.T) {
	const secret = "sk-test-opencode-key"
	fake := &fakeOpenCode{status: 200, body: `{"choices":[{"message":{"content":"OK"}}]}`}
	home := setupOpenCode(t, fake, secret)
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(Options{Workspace: t.TempDir(), Manifest: manifest, Trusted: true, SessionID: "s-1"})
	m = pickOpenCode(t, m, "/provider")
	if m.opt.Provider == nil {
		t.Fatalf("provider not connected: %v", m.transcript)
	}
	if m.opt.WorkerModel != "kimi-k2.7-code" || fake.model != "kimi-k2.7-code" {
		t.Errorf("model = %q (sent %q), want kimi-k2.7-code", m.opt.WorkerModel, fake.model)
	}
	if fake.auth != "Bearer "+secret || fake.session != "s-1" {
		t.Errorf("ping headers auth=%q session=%q", fake.auth, fake.session)
	}
	if !strings.Contains(m.View(), "connected OpenCode Go") {
		t.Errorf("View() missing connected line:\n%s", m.View())
	}
	if strings.Contains(m.View(), secret) {
		t.Error("api key visible in the view")
	}
	if got := config.LastProvider(); got != "opencode-go" {
		t.Errorf("LastProvider() = %q, want opencode-go", got)
	}
	catalog, err := os.ReadFile(filepath.Join(home, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(catalog), secret) {
		t.Fatalf("secret stored in catalog: %s", catalog)
	}
}

func TestProvidersShowsConnectError(t *testing.T) {
	fake := &fakeOpenCode{status: 401, body: `{"error":{"message":"Invalid API key."}}`}
	setupOpenCode(t, fake, "sk-bad")
	m := NewModel(Options{Workspace: t.TempDir(), Trusted: true})
	m = pickOpenCode(t, m, "/providers")
	if m.opt.Provider != nil {
		t.Fatal("provider connected despite 401")
	}
	view := m.View()
	if !strings.Contains(view, "could not connect to OpenCode Go") || !strings.Contains(view, "Invalid API key.") {
		t.Errorf("View() missing connect error:\n%s", view)
	}
	if config.LastProvider() != "" {
		t.Error("failed connection was saved as last provider")
	}
}

func TestReconnectRestoresModel(t *testing.T) {
	fake := &fakeOpenCode{status: 200, body: `{}`}
	setupOpenCode(t, fake, "sk-test")
	if err := config.SaveLastProvider("opencode-go"); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveLastModel("gpt-6-luna"); err != nil {
		t.Fatal(err)
	}
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(Options{Workspace: t.TempDir(), Manifest: manifest, Trusted: true, AutoConnect: "opencode-go"})
	updated, _ := m.Update(m.startCmd())
	got := updated.(Model)
	if got.opt.WorkerModel != "gpt-6-luna" {
		t.Fatalf("WorkerModel = %q, want gpt-6-luna", got.opt.WorkerModel)
	}
	md, err := got.opt.Manifest.ResolveRole("worker")
	if err != nil {
		t.Fatal(err)
	}
	if md.Model != "gpt-6-luna" {
		t.Fatalf("worker model = %q, want gpt-6-luna", md.Model)
	}
	if !strings.Contains(got.transcriptText(), "gpt-6-luna") {
		t.Fatalf("transcript %q", got.transcriptText())
	}
}

func TestAutoConnectLastProvider(t *testing.T) {
	fake := &fakeOpenCode{status: 200, body: `{}`}
	setupOpenCode(t, fake, "sk-test")
	m := NewModel(Options{Workspace: t.TempDir(), Trusted: true, AutoConnect: "opencode-go"})
	if m.startCmd == nil {
		t.Fatal("NewModel did not schedule a reconnect")
	}
	updated, _ := m.Update(m.startCmd())
	if updated.(Model).opt.Provider == nil {
		t.Fatalf("auto-connect failed: %v", updated.(Model).transcript)
	}
}

func TestSlashExit(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.input.SetValue("/exit")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(/exit) cmd = nil, want quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("Update(/exit) msg = %T, want tea.QuitMsg", cmd())
	}
}

func TestSlashHelp(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.input.SetValue("/help")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if !strings.Contains(model.transcriptText(), "/abort") {
		t.Fatalf("%v", model.transcriptText())
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
	tests := map[string]string{"t": "trust", "o": "trust_once", "d": "dont_trust", "esc": "dont_trust"}
	for key, want := range tests {
		modal := NewTrustModal("/ws")
		modal.Press(key)
		if modal.Decision != want {
			t.Errorf("Press(%q) decision = %q, want %q", key, modal.Decision, want)
		}
	}
}

func untrustedFixture(t *testing.T) Model {
	t.Helper()
	t.Setenv("MUXON_HOME", t.TempDir())
	t.Setenv("MUXON_TRUST", "")
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(Options{
		Workspace: dir,
		SessionID: "trust-session",
		Provider:  scripted.New(director.M0ScriptedResponses()),
		Manifest:  manifest,
	})
	if m.modal == nil || m.modal.Kind != "trust" {
		t.Fatal("untrusted workspace did not open the trust prompt on start")
	}
	if !strings.Contains(m.View(), "Trust this workspace?") {
		t.Fatalf("View() missing trust prompt:\n%s", m.View())
	}
	return m
}

func TestTrustOnceRunsPendingGoal(t *testing.T) {
	m := untrustedFixture(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if !strings.Contains(m.View(), "Workspace not trusted") {
		t.Fatalf("View() missing not-trusted note:\n%s", m.View())
	}
	m.input.SetValue("fix the failing test")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil || m.modal == nil {
		t.Fatal("goal ran in an untrusted workspace instead of asking for trust")
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if cmd == nil {
		t.Fatal("trusting did not start the pending goal")
	}
	m = drainGoal(t, updated.(Model), cmd)
	if m.LastOutcome().State != director.StateDone {
		t.Fatalf("pending goal outcome %+v", m.LastOutcome())
	}
	if security.IsTrustedRoot(m.opt.Workspace) {
		t.Error("trust once was saved to the trust store")
	}
}

func TestTrustSavesWorkspace(t *testing.T) {
	m := untrustedFixture(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = updated.(Model)
	if !m.opt.Trusted || !security.IsTrustedRoot(m.opt.Workspace) {
		t.Fatal("trust was not applied and saved")
	}
	if strings.Contains(m.View(), "untrusted") {
		t.Errorf("footer still says untrusted:\n%s", m.View())
	}
}

func TestInterruptSecondCancels(t *testing.T) {
	m := sessionModel(t.TempDir())
	now := time.Now()
	m.Interrupt(now)
	m.Interrupt(now.Add(time.Second))
	if m.LastOutcome().ExitCode != 130 {
		t.Fatalf("%+v", m.LastOutcome())
	}
}

func TestScriptedFixtureDone(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
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
	model := drainGoal(t, updated.(Model), cmd)
	if model.LastOutcome().State != director.StateDone {
		t.Fatalf("%+v\n%s", model.LastOutcome(), model.View())
	}
	if view := model.View(); !strings.Contains(view, "Edit sum.go") || !strings.Contains(view, "⎿  passed") {
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
	model = drainGoal(t, updated.(Model), cmd)
	if model.LastOutcome().State != director.StateDone {
		t.Fatalf("second goal %+v", model.LastOutcome())
	}
}

func TestAskUserShowsQuestionAndSendsAnswer(t *testing.T) {
	askArgs, _ := json.Marshal(map[string]any{"question": "Which file?", "choices": []string{"sum.go", "main.go"}})
	finishArgs, _ := json.Marshal(map[string]string{"summary": "picked sum.go"})
	t.Setenv("MUXON_HOME", t.TempDir())
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	m := sessionModel(testutil.CopyFixture(t, "go-failing-test"))
	m.opt.Manifest = manifest
	m.opt.Provider = scripted.New([]provider.Response{
		{ToolCalls: []provider.ToolCall{{ID: "1", Name: "ask_user", Arguments: askArgs}}, FinishReason: "tool_calls"},
		{ToolCalls: []provider.ToolCall{{ID: "2", Name: "finish", Arguments: finishArgs}}, FinishReason: "tool_calls"},
	})
	m.input.SetValue("hello")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	answered := false
	for i := 0; i < 200 && cmd != nil; i++ {
		msg := cmd()
		updated, cmd = model.Update(msg)
		model = updated.(Model)
		if _, ok := msg.(askUserMsg); ok {
			view := model.View()
			if !strings.Contains(view, "Which file?") || !strings.Contains(view, "1. sum.go") || !strings.Contains(view, "waiting for your answer") {
				t.Fatalf("question not shown:\n%s", view)
			}
			model.input.SetValue("1")
			updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			model = updated.(Model)
			answered = true
			cmd = listen(model.events)
		}
		if _, ok := msg.(goalDone); ok {
			break
		}
	}
	if !answered {
		t.Fatal("ask_user never reached the TUI")
	}
	if model.LastOutcome().State != director.StateDone {
		t.Fatalf("%+v\n%s", model.LastOutcome(), model.transcriptText())
	}
	if view := model.View(); !strings.Contains(view, "picked sum.go") {
		t.Fatalf("finish summary missing:\n%s", view)
	}
}

func TestPausedOutcomeOffersContinue(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.appendLine("[paused] tool_rounds_exhausted")
	view := m.View()
	for _, want := range []string{"paused", "tool round limit reached", "/continue to keep going"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q:\n%s", want, view)
		}
	}
	updated, _ := m.submit("/continue")
	if !strings.Contains(updated.(Model).transcriptText(), "nothing to continue") {
		t.Fatalf("transcript: %s", updated.(Model).transcriptText())
	}
}

func TestContinueAfterPauseKeepsHistory(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, err := config.LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	readArgs, _ := json.Marshal(map[string]any{"path": "sum.go", "start_line": 1, "end_line": 3})
	m := NewModel(Options{
		Workspace: dir,
		SessionID: "tui-continue",
		Trusted:   true,
		Provider: scripted.New([]provider.Response{
			{ToolCalls: []provider.ToolCall{{ID: "1", Name: "read", Arguments: readArgs}}, FinishReason: "tool_calls"},
			{Text: "Continuing the web search work."},
		}),
		Manifest:    manifest,
		Version:     "test",
		WorkerModel: "m0-worker",
		Permission:  "edit",
	})
	m.history = []provider.Message{{Role: "user", Content: "add web search"}}
	m.lastOutcome = director.Outcome{State: director.StatePaused, Reason: "tool_rounds_exhausted"}
	updated, cmd := m.submit("/continue")
	if cmd == nil {
		t.Fatal("expected continue to start a goal")
	}
	model := drainGoal(t, updated.(Model), cmd)
	joined := model.transcriptText()
	if !strings.Contains(joined, continueGoal) || !strings.Contains(joined, "Continuing the web search work.") {
		t.Fatalf("transcript: %s", joined)
	}
	var hist strings.Builder
	for _, msg := range model.history {
		hist.WriteString(msg.Content)
		hist.WriteByte('\n')
	}
	if !strings.Contains(hist.String(), "add web search") || !strings.Contains(hist.String(), continueGoal) {
		t.Fatalf("history: %s", hist.String())
	}
}

func drainGoal(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()
	sawTool := false
	for i := 0; i < 200; i++ {
		if cmd == nil {
			t.Fatal("goal stopped before goalDone")
		}
		msg := cmd()
		if msg == nil {
			t.Fatal("event channel closed before goalDone")
		}
		if ev, ok := msg.(liveEvent); ok && ev.ev.Type == "tool.call" {
			sawTool = true
		}
		if done, ok := msg.(goalDone); ok {
			if !sawTool {
				t.Fatalf("tool line did not arrive before goalDone: outcome %+v transcript %v", done.out, model.transcript)
			}
			updated, _ := model.Update(msg)
			return updated.(Model)
		}
		updated, next := model.Update(msg)
		model = updated.(Model)
		cmd = next
	}
	t.Fatal("goal did not finish")
	return Model{}
}
