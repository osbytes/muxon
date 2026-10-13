package contextengine

import (
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

func TestBuildTurnIncludesToolsWhenNative(t *testing.T) {
	out := BuildTurn(TurnInput{
		Goal:         "fix test",
		ToolProtocol: "native",
		ToolSpecs: []tools.Spec{
			{Name: "read", Description: "read", Parameters: []byte(`{}`)},
		},
	})
	if len(out.Tools) != 1 || out.Tools[0].Name != "read" {
		t.Fatalf("tools: %+v", out.Tools)
	}
}
