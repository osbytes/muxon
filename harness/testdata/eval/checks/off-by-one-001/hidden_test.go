package seq

import "testing"

func TestLast(t *testing.T) {
	if Last(5) != 4 {
		t.Fatalf("Last(5) = %d", Last(5))
	}
}
