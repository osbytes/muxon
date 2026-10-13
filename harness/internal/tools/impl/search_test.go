package impl

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
	"github.com/PLACEHOLDER/thee/harness/internal/workspace"
)

func TestSearchFindsSum(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not installed")
	}
	dir := testutil.CopyFixture(t, "go-failing-test")
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := &tools.Env{Workspace: ws, Trusted: true}
	args, _ := json.Marshal(map[string]string{"pattern": "Sum"})
	res, err := NewSearch().Run(context.Background(), env, args)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !strings.Contains(res.Text, "Sum") {
		t.Fatalf("got %+v", res)
	}
}
