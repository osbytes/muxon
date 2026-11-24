package router

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider/openai"
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

func TestOpenCodeProviderSetsSessionHeader(t *testing.T) {
	t.Setenv("TEST_API_KEY", "secret-not-logged")
	m := &config.Manifest{
		Providers: map[string]config.ProviderDef{
			"opencode-go": {BaseURL: "https://opencode.ai/zen/go/v1", APIKeyEnv: "TEST_API_KEY"},
		},
		Roles: map[string]config.RoleDef{
			"worker": {ModelID: "kimi"},
		},
		Models: []config.ModelDef{{
			ID: "kimi", Provider: "opencode-go", Model: "kimi-k2.7-code",
		}},
	}
	p, _, err := NewProviderForRole(m, "worker", "")
	if err != nil {
		t.Fatal(err)
	}
	client, ok := p.(*openai.Client)
	if !ok {
		t.Fatalf("provider type %T", p)
	}
	if client.Headers["x-opencode-session"] == "" {
		t.Fatal("missing x-opencode-session")
	}
}
