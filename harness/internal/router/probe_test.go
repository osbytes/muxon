package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeChatWritesAcceptedSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{}","tool_calls":[{"id":"1","type":"function","function":{"name":"ping","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`))
	}))
	defer srv.Close()
	sn, err := ProbeChat(context.Background(), srv.URL, "", "probe-model")
	if err != nil {
		t.Fatal(err)
	}
	if sn.StructuredOutput != "json_schema" || sn.ToolProtocol != "native" {
		t.Fatalf("%+v", sn)
	}
	t.Setenv("THEE_HOME", t.TempDir())
	if err := WriteSnippet(sn); err != nil {
		t.Fatal(err)
	}
}
