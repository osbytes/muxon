package patch

import (
	"testing"
)

func TestSearchReplaceExact(t *testing.T) {
	out, strategy, err := ApplySearchReplaceExact("foo\nbar\n", []Replacement{
		{Search: "bar", Replace: "baz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strategy != "exact" || string(out) != "foo\nbaz\n" {
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
	out, strategy, err := ApplyUDiffExact(content, diff)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != "exact" {
		t.Fatalf("strategy %q", strategy)
	}
	if string(out) != "line1\nline2fixed\nline3\n" {
		t.Fatalf("got %q", out)
	}
}
