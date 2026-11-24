package scripted

import (
	"context"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

func TestScriptedDrainsInOrder(t *testing.T) {
	p := New([]provider.Response{
		{Text: "one"},
		{Text: "two"},
	})
	ctx := context.Background()
	r1, err := p.Complete(ctx, provider.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if r1.Text != "one" {
		t.Fatalf("r1: %q", r1.Text)
	}
	r2, err := p.Complete(ctx, provider.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Text != "two" {
		t.Fatalf("r2: %q", r2.Text)
	}
}
