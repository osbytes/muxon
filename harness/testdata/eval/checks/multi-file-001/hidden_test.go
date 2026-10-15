package calc

import "testing"

func TestDouble(t *testing.T) {
	if Double(3) != 6 {
		t.Fatalf("Double(3) = %d", Double(3))
	}
	if Add(1, 2) != 3 {
		t.Fatalf("Add(1, 2) = %d", Add(1, 2))
	}
}
