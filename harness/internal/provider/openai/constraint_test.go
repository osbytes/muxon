package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/toolcall"
)

func TestConstraintDowngradeRetry(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid response_format json_schema"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tool\":\"read\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "k")
	req := provider.Request{
		ModelID:        "m",
		ResponseFormat: toolcall.Step1Schema([]string{"read"}),
	}
	resp, err := c.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if attempts < 2 {
		t.Fatal("expected retry")
	}
	if resp.Text == "" {
		t.Fatal("empty")
	}
}
