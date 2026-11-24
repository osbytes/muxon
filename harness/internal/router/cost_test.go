package router

import (
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

func TestCachedTokensPricedSeparately(t *testing.T) {
	r := Rates{InputPerM: 1.0, CachedInputPerM: 0.1, OutputPerM: 2.0}
	u := provider.Usage{InputTokens: 1000, CachedTokens: 800, OutputTokens: 100}
	got, ok := CostUSD(r, u)
	if !ok {
		t.Fatal("expected cost")
	}
	want := (200*1.0 + 800*0.1 + 100*2.0) / 1e6
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
