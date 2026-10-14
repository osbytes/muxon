package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMatchSuggestions(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "hello"},
		{in: "/", want: []string{"/providers", "/model", "/help", "/trust", "/cost", "/escalate", "/undo", "/abort", "/exit"}},
		{in: "/ex", want: []string{"/exit"}},
		{in: "/qu", want: []string{"/exit"}},
		{in: "/co", want: []string{"/cost"}},
		{in: "/con", want: []string{"/providers"}},
		{in: "/providers ", want: []string{"opencode-go", "deepinfra"}},
		{in: "/providers d", want: []string{"deepinfra"}},
		{in: "/connect o", want: []string{"opencode-go"}},
		{in: "/cost "},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := labels(matchSuggestions(tt.in))
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("matchSuggestions(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSuggestionMenuFiltersView(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.input.SetValue("/co")
	menu := m.suggestionView(m.contentWidth())
	if !strings.Contains(menu, "/cost") || !strings.Contains(menu, "session cost") || !strings.Contains(menu, "❯") {
		t.Fatalf("menu %q", menu)
	}
	if strings.Contains(menu, "/help") {
		t.Fatalf("menu listed /help: %q", menu)
	}
	if !strings.Contains(m.View(), "tab complete") {
		t.Fatalf("view missing complete hint:\n%s", m.View())
	}
}

func TestSuggestionTabAndEnter(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.input.SetValue("/")
	var updated tea.Model
	for range 2 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.input.Value() != "/help" {
		t.Fatalf("tab inserted %q, want /help", m.input.Value())
	}

	m.input.SetValue("/co")
	m.completeIndex = 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !strings.Contains(m.transcriptText(), "rounds") {
		t.Fatalf("enter did not run /cost: %q", m.transcriptText())
	}
}

func TestSuggestionEscSendsTypedText(t *testing.T) {
	m := sessionModel(t.TempDir())
	m.input.SetValue("/co")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.suggestionView(m.contentWidth()) != "" {
		t.Fatal("menu stayed open after esc")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !strings.Contains(m.transcriptText(), "> /co") {
		t.Fatalf("sent %q, want the typed text", m.transcriptText())
	}
	m.cancelRun()
}

func labels(list []suggestion) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.label
	}
	return out
}
