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
