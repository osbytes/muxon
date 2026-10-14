package replay

import (
	"context"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func TestHashMismatchDiff(t *testing.T) {
	p := New([]Step{{
		Hash:     "deadbeef",
		Texts:    []string{"user: expected"},
		Response: provider.Response{Text: "nope"},
	}})
	_, err := p.Complete(context.Background(), provider.Request{
		Messages: []provider.Message{{Role: "user", Content: "actual"}},
	})
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") || !strings.Contains(err.Error(), "actual") {
		t.Fatalf("%v", err)
	}
}
