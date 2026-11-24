package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

func TestStreamRecordedSSE(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "testdata", "provider", "openai_sse_simple.txt")
	body, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	stream, err := c.Stream(context.Background(), provider.Request{ModelID: "m"})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := stream.Next()
	if err != nil {
		t.Fatal(err)
	}
	_ = stream.Close()
	if resp.Text != "hello" {
		t.Fatalf("text: %q", resp.Text)
	}
	if resp.Usage.InputTokens != 3 || resp.Usage.OutputTokens != 1 {
		t.Fatalf("usage: %+v", resp.Usage)
	}
}

func TestCompleteJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), provider.Request{ModelID: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "ok" {
		t.Fatalf("text: %q", resp.Text)
	}
}
