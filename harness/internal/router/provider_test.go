package router

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func TestWorkerProviderSendsBoundModelID(t *testing.T) {
	t.Setenv("TEST_API_KEY", "secret-not-logged")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Model != "sku/worker-1" {
			t.Errorf("model %q", req.Model)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()

	m := &config.Manifest{
		Providers: map[string]config.ProviderDef{
			"p": {BaseURL: srv.URL, APIKeyEnv: "TEST_API_KEY"},
		},
		Roles: map[string]config.RoleDef{
			"worker": {ModelID: "w"},
		},
		Models: []config.ModelDef{{
			ID: "w", Provider: "p", Model: "sku/worker-1",
		}},
	}
	p, b, err := NewProviderForRole(m, "worker", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Model.Model != "sku/worker-1" {
		t.Fatal(b.Model.Model)
	}
	_, err = p.Complete(context.Background(), provider.Request{ModelID: b.Model.Model, Messages: []provider.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
}
