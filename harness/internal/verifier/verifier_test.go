package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestVerifierOnFixture(t *testing.T) {
	dir := testutil.CopyFixture(t, "go-failing-test")
	ctx := context.Background()
	cmd, ok := ResolveCommand(dir)
	if !ok {
		t.Fatal("expected go test command")
	}
	res, err := Run(ctx, dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed {
		t.Fatal("expected failing tests before fix")
	}
	// apply fix
	bug := filepath.Join(dir, "sum.go")
	fixed := []byte(`package sum

func Sum(a, b int) int {
	return a + b
}
`)
	if err := os.WriteFile(bug, fixed, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = Run(ctx, dir, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("expected pass after fix: %s", res.Excerpt)
	}
}
