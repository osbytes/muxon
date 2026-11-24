package sum

import "testing"

func TestSumHidden(t *testing.T) {
	if Sum(2, 3) != 5 {
		t.Fatalf("Sum(2, 3) = %d", Sum(2, 3))
	}
	if Sum(-1, 4) != 3 {
		t.Fatalf("Sum(-1, 4) = %d", Sum(-1, 4))
	}
}
