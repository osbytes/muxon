package index

import (
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestRepoMapRanksTestSymbol(t *testing.T) {
	dir := testutil.CopyFixture(t, "go-failing-test")
	pol := security.NewPolicy()
	ign, err := security.LoadIgnore(dir, pol)
	if err != nil {
		t.Fatal(err)
	}
	syms, err := BuildGoRepoMap(dir, "fix TestSum failing", ign, pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) == 0 {
		t.Fatal("no symbols")
	}
	top := syms[0]
	if !strings.Contains(top.Path, "test") && top.Name != "TestSum" {
		found := false
		for _, s := range syms[:3] {
			if s.Name == "TestSum" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("TestSum not in top symbols: %+v", syms[:5])
		}
	}
}
