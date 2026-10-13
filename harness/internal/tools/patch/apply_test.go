package patch

import (
	"testing"
)

func TestSearchReplaceExact(t *testing.T) {
	out, strat, err := ApplySearchReplaceExact("foo\nbar\n", []Replacement{
		{Search: "bar", Replace: "baz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strat != "exact" || string(out) != "foo\nbaz\n" {
		t.Fatalf("got %q", out)
	}
}

func TestUDiffExact(t *testing.T) {
	content := "line1\nline2\nline3\n"
	diff := `--- a
+++ b
@@ -2,1 +2,1 @@
-line2
+line2fixed
`
	out, strat, err := ApplyUDiffExact(content, diff)
	if err != nil {
		t.Fatal(err)
	}
	if strat != "exact" {
		t.Fatalf("strategy %q", strat)
	}
	if string(out) != "line1\nline2fixed\nline3\n" {
		t.Fatalf("got %q", out)
	}
}
