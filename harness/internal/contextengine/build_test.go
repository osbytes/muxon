package contextengine

import (
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

func TestPrefixHashStableAcrossPhase(t *testing.T) {
	base := TurnInput{
		Goal:          "fix test",
		ToolProtocol:  "native",
		PromptVariant: VariantCompact,
		WorkspaceRoot: t.TempDir(),
		ToolSpecs: []tools.Spec{
			{Name: "read", Description: "read"},
		},
		ContextWindow: 32768,
	}
	h1 := PrefixHashForSession(base)
	base.Phase = tools.PhasePlanning
	h2 := PrefixHashForSession(base)
	if h1 != h2 {
		t.Fatalf("phase change should not affect prefix hash: %s vs %s", h1, h2)
	}
}

func TestBuildTurnKeepsGoalAfterHistory(t *testing.T) {
	out := BuildTurn(TurnInput{
		Goal: "where is the loop?",
		History: []provider.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "Hi."},
			{Role: "tool", Name: "read", Content: "package director"},
		},
		ToolProtocol:  "native",
		ContextWindow: 32768,
	})
	var saw bool
	for _, m := range out.Messages {
		if m.Role == "user" && m.Content == "where is the loop?" {
			saw = true
		}
	}
	if !saw {
		t.Fatal("BuildTurn() messages missing current goal")
	}
	if strings.Count(messageText(out.Messages), "where is the loop?") != 1 {
		t.Fatalf("BuildTurn() duplicated the goal")
	}
}

func messageText(msgs []provider.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestBuildTurnShowsRoundsAndActiveTools(t *testing.T) {
	specs := []tools.Spec{
		{Name: "read", Description: "read"},
		{Name: "write", Description: "write"},
		{Name: "list", Description: "list"},
	}
	below := BuildTurn(TurnInput{
		Goal:          "fix it",
		ToolProtocol:  "native",
		ToolSpecs:     specs,
		ContextWindow: 32768,
		Phase:         tools.PhaseEditing,
		ToolRounds:    31,
		MaxToolRounds: 40,
	})
	text := messageText(below.Messages)
	if !strings.Contains(text, "Tool rounds: 31/40") {
		t.Fatalf("tail missing rounds: %s", text)
	}
	if strings.Contains(text, "near the tool round limit") {
		t.Fatalf("wrap-up appeared before the threshold: %s", text)
	}
	if !strings.Contains(text, "Active: read, write") {
		t.Fatalf("active tools = %s", text)
	}
	if strings.Contains(text, "list") && strings.Contains(text, "Active: read, write, list") {
		t.Fatalf("planning-only list tool listed in editing: %s", text)
	}
	above := BuildTurn(TurnInput{
		Goal:          "fix it",
		ToolProtocol:  "native",
		ToolSpecs:     specs,
		ContextWindow: 32768,
		Phase:         tools.PhaseEditing,
		ToolRounds:    32,
		MaxToolRounds: 40,
	})
	if !strings.Contains(messageText(above.Messages), wrapUpNudge) {
		t.Fatal("wrap-up missing at 32/40")
	}
	if wrapUpAt(10) != 5 {
		t.Fatalf("wrapUpAt(10) = %d, want 5", wrapUpAt(10))
	}
}

func TestPrefixHashStableSameSession(t *testing.T) {
	dir := t.TempDir()
	in := TurnInput{
		Goal:          "fix TestSum",
		ToolProtocol:  "native",
		WorkspaceRoot: dir,
		ToolSpecs:     []tools.Spec{{Name: "read", Description: "r"}},
		ContextWindow: 32768,
	}
	h1 := PrefixHashForSession(in)
	in.Phase = tools.PhasePlanning
	h2 := PrefixHashForSession(in)
	if h1 != h2 {
		t.Fatalf("expected stable prefix in session: %s vs %s", h1, h2)
	}
}
