package director

import (
	"context"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/testutil"
)

func TestUntrustedHeadlessExit3(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	dir := testutil.CopyFixture(t, "go-failing-test")
	manifest, _ := config.LoadEmbeddedManifest()
	out, err := Run(context.Background(), Config{
		Goal: "fix", Workspace: dir, Trusted: false, Provider: scripted.New(M0ScriptedResponses()), Manifest: manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ExitCode != 3 || out.Reason != "workspace_untrusted" {
		t.Fatalf("%+v", out)
	}
}
