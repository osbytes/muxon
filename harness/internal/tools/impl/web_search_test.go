package impl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
)

func TestWebSearchReturnsSearchResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "Go context" {
			t.Errorf("query = %q, want %q", got, "Go context")
		}
		_, _ = w.Write([]byte(`<div class="result"><a rel="nofollow" class="result__a" href="https://example.com/docs">Go &amp; Context</a><a class="result__snippet">A useful <b>guide</b>.</a></div>`))
	}))
	defer server.Close()

	tool := &webSearchTool{client: server.Client(), endpoint: server.URL}
	args, err := json.Marshal(webSearchArgs{Query: "Go context", MaxResults: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Run(context.Background(), &tools.Env{Trusted: true}, args)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Go & Context", "https://example.com/docs", "A useful guide.", "untrusted"} {
		if !strings.Contains(result.Text, want) {
			t.Errorf("result missing %q: %s", want, result.Text)
		}
	}
	if !result.OK {
		t.Fatalf("web search failed: %s", result.Text)
	}
}

func TestWebSearchRejectsEmptyQuery(t *testing.T) {
	tool := &webSearchTool{client: http.DefaultClient, endpoint: "https://example.com"}
	result, err := tool.Run(context.Background(), &tools.Env{Trusted: true}, json.RawMessage(`{"query":"  "}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || !strings.Contains(result.Text, "query must not be empty") {
		t.Fatalf("got %+v", result)
	}
}

func TestNormalizeWebSearchURL(t *testing.T) {
	got, ok := normalizeWebSearchURL("https://duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage")
	if !ok || got != "https://example.com/page" {
		t.Fatalf("normalizeWebSearchURL() = %q, %v", got, ok)
	}
	if _, ok := normalizeWebSearchURL("javascript:alert(1)"); ok {
		t.Fatal("accepted non-http search URL")
	}
}
