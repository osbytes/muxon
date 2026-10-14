package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
)

func TestModelCommandSwitchesWorker(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	m := sessionModel(t.TempDir())
	m.modelCommand("qwen3-coder")
	if m.opt.WorkerModel != "qwen3-coder" {
		t.Fatalf("WorkerModel = %q, want qwen3-coder", m.opt.WorkerModel)
	}
	md, err := m.opt.Manifest.ResolveRole("worker")
	if err != nil {
		t.Fatal(err)
	}
	if md.Model != "qwen3-coder" {
		t.Fatalf("worker model = %q, want qwen3-coder", md.Model)
	}
	if !strings.Contains(m.transcriptText(), "model qwen3-coder") {
		t.Fatalf("transcript %q", m.transcriptText())
	}
	if md.Capabilities.ToolProtocol == "" {
		t.Fatal("worker capabilities were not filled")
	}
	if config.LastModel() != "qwen3-coder" {
		t.Fatalf("LastModel() = %q, want qwen3-coder", config.LastModel())
	}
}

func TestModelCommandRejectsUnknownWhenListed(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.modelsAuthoritative = true
	m.modelChoices = []string{"kimi-k2.7-code"}
	m.modelCommand("nope")
	if m.opt.WorkerModel == "nope" {
		t.Fatal("switched to an unknown model")
	}
	if !strings.Contains(m.transcriptText(), "unknown model nope") {
		t.Fatalf("transcript %q", m.transcriptText())
	}
}

func TestModelSuggestionsFilter(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.modelChoices = []string{"kimi-k2.7-code", "qwen3-coder"}
	m.opt.WorkerModel = "kimi-k2.7-code"
	m.input.SetValue("/model q")
	menu := m.suggestionView(m.contentWidth())
	if !strings.Contains(menu, "qwen3-coder") {
		t.Fatalf("menu %q", menu)
	}
	if strings.Contains(menu, "kimi-k2.7-code") {
		t.Fatalf("menu listed the current model: %q", menu)
	}

	m.input.SetValue("/model ")
	menu = m.suggestionView(m.contentWidth())
	if !strings.Contains(menu, "kimi-k2.7-code") || !strings.Contains(menu, "current") {
		t.Fatalf("menu %q", menu)
	}
}

func TestModelPickerSelects(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	m := sessionModel(t.TempDir())
	m.modelChoices = []string{"kimi-k2.7-code", "qwen3-coder"}
	m.modelsAuthoritative = true
	m.modelPicker = true
	m.opt.WorkerModel = "kimi-k2.7-code"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.modelPicker {
		t.Fatal("picker stayed open")
	}
	if m.opt.WorkerModel != "qwen3-coder" {
		t.Fatalf("WorkerModel = %q, want qwen3-coder", m.opt.WorkerModel)
	}
	if !strings.Contains(m.View(), "qwen3-coder") {
		t.Fatalf("footer missing model:\n%s", m.View())
	}
}

func TestModelCommandOpensPicker(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.input.SetValue("/model")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.modelPicker {
		t.Fatal("picker did not open")
	}
	if !strings.Contains(m.View(), "Switch model") {
		t.Fatalf("view %s", m.View())
	}
}
