package director

import (
	"context"
	"testing"
	"time"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

type blockingProvider struct{}

func (blockingProvider) Complete(ctx context.Context, req provider.Request) (provider.Response, error) {
	select {
	case <-ctx.Done():
		return provider.Response{}, ctx.Err()
	case <-time.After(5 * time.Second):
		return provider.Response{}, nil
	}
}

func (blockingProvider) Stream(ctx context.Context, req provider.Request) (provider.Stream, error) {
	return nil, context.Canceled
}

func TestDirectorCancelExit130(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manifest, _ := config.LoadEmbeddedManifest()
	out, err := Run(ctx, Config{
		Goal:        "fix",
		Workspace:   t.TempDir(),
		Trusted:     true,
		TheeVersion: "test",
		Provider:    blockingProvider{},
		Manifest:    manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ExitCode != 130 || out.State != StateCancelled {
		t.Fatalf("outcome %+v", out)
	}
}
