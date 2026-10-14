package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "ok", status: 200, body: `{"choices":[{"message":{"content":"OK"}}]}`},
		{name: "bad key", status: 401, body: `{"error":{"message":"Invalid API key."}}`, wantErr: "Invalid API key."},
		{name: "plain body", status: 500, body: `upstream down`, wantErr: "upstream down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAuth, gotSession string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotSession = r.Header.Get("x-opencode-session")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "sk-test")
			c.Headers = map[string]string{"x-opencode-session": "s1"}
			err := c.Ping(context.Background(), "kimi-k2.7-code")
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Ping() = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Ping() = %v, want error containing %q", err, tt.wantErr)
			}
			if gotAuth != "Bearer sk-test" || gotSession != "s1" {
				t.Errorf("Ping() headers auth=%q session=%q, want Bearer sk-test and s1", gotAuth, gotSession)
			}
		})
	}
}
