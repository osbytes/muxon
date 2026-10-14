package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListModels(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr string
	}{
		{
			name:   "data",
			status: 200,
			body:   `{"data":[{"id":"qwen3-coder"},{"id":"kimi-k2.7-code"},{"id":"kimi-k2.7-code"}]}`,
			want:   "kimi-k2.7-code,qwen3-coder",
		},
		{
			name:   "models",
			status: 200,
			body:   `{"models":[{"name":"glm-5"}]}`,
			want:   "glm-5",
		},
		{
			name:    "unauthorized",
			status:  401,
			body:    `{"error":{"message":"bad key"}}`,
			wantErr: "bad key",
		},
		{
			name:    "empty",
			status:  200,
			body:    `{"data":[]}`,
			wantErr: "empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				if r.URL.Path != "/models" {
					t.Errorf("path = %s, want /models", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "sk-test")
			names, err := c.ListModels(context.Background())
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ListModels() = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("ListModels() = %v, want error containing %q", err, tt.wantErr)
			}
			if strings.Join(names, ",") != tt.want {
				t.Errorf("ListModels() = %v, want %v", names, tt.want)
			}
			if gotAuth != "Bearer sk-test" {
				t.Errorf("Authorization = %q, want Bearer sk-test", gotAuth)
			}
		})
	}
}
