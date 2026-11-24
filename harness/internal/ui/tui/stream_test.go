package tui

import (
	"fmt"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

type fakeStream struct {
	parts []provider.Response
	i     int
}

func (f *fakeStream) Next() (provider.Response, error) {
	if f.i >= len(f.parts) {
		return provider.Response{}, fmt.Errorf("EOF")
	}
	r := f.parts[f.i]
	f.i++
	return r, nil
}

func (f *fakeStream) Close() error { return nil }

func TestCollectText(t *testing.T) {
	s := &fakeStream{parts: []provider.Response{
		{Text: "hello "},
		{Text: "world", FinishReason: "stop"},
	}}
	got, err := CollectText(s)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("%q", got)
	}
}
