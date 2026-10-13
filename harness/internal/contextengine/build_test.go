package contextengine

import (
	"testing"

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
