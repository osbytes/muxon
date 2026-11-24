package patch

import "testing"

func TestWhitespaceLadder(t *testing.T) {
	content := "foo\n  bar\n"
	reps := []Replacement{{Search: "foo\nbar", Replace: "foo\nbaz"}}
	_, _, err := ApplySearchReplaceExact(content, reps)
	if err == nil {
		t.Fatal("exact should fail on whitespace mismatch")
	}
	out, strategy, err := ApplySearchReplace(content, reps)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != "whitespace" {
		t.Fatalf("strategy %q", strategy)
	}
	if !contains(string(out), "baz") {
		t.Fatalf("got %q", out)
	}
}

func TestRejectAmbiguous(t *testing.T) {
	content := "x y x y"
	reps := []Replacement{{Search: "a", Replace: "b"}}
	_, _, err := ApplySearchReplace(content, reps)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
