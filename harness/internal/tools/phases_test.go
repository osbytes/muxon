package tools

import "testing"

func TestPatchRejectedInPlanning(t *testing.T) {
	r := NewRegistry()
	if err := r.Check("patch", PhasePlanning); err == nil {
		t.Fatal("expected error")
	}
	if err := r.Check("patch", PhaseEditing); err != nil {
		t.Fatal(err)
	}
}

func TestWebSearchAllowedInBothPhases(t *testing.T) {
	for _, phase := range []string{PhasePlanning, PhaseEditing} {
		if err := CheckPhase(phase, "web_search"); err != nil {
			t.Errorf("web_search rejected in %s: %v", phase, err)
		}
	}
}
