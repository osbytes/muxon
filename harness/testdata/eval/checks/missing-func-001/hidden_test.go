package greet

import "testing"

func TestGreeter(t *testing.T) {
	if Greeter() != "hi" {
		t.Fatalf("%q", Greeter())
	}
}
