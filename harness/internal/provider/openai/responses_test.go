package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

func TestResponsesEndpointForOpenCodeGo(t *testing.T) {
	c := NewClient("https://opencode.ai/zen/go/v1", "sk")
	if got := c.responsesEndpoint(); got != openCodeGoResponsesURL {
		t.Fatalf("responsesEndpoint() = %q, want %q", got, openCodeGoResponsesURL)
	}
}

func TestCompleteRoutesLunaToResponses(t *testing.T) {
	var gotPath, gotModel, gotToolType string
	var gotStore bool
	var gotInput []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body struct {
			Model string           `json:"model"`
			Store bool             `json:"store"`
			Input []map[string]any `json:"input"`
			Tools []struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"tools"`
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"store":false`) {
			t.Errorf("request missing store:false: %s", b)
		}
		if err := json.Unmarshal(b, &body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotModel = body.Model
		gotStore = body.Store
		gotInput = body.Input
		if len(body.Tools) > 0 {
			gotToolType = body.Tools[0].Type + ":" + body.Tools[0].Name
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output":[
				{"type":"reasoning","id":"rs_1"},
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hi"}]},
				{"type":"function_call","call_id":"call_1","name":"read","arguments":"{\"path\":\"a.go\"}"}
			],
			"usage":{"input_tokens":3,"output_tokens":4,"output_tokens_details":{"reasoning_tokens":1}}
		}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sk-test")
	resp, err := c.Complete(context.Background(), provider.Request{
		ModelID: "gpt-6-luna",
		Messages: []provider.Message{
			{Role: "system", Content: "be brief"},
			{Role: "user", Content: "Hello"},
		},
		Tools: []provider.ToolDef{{
			Name:       "read",
			Parameters: json.RawMessage(`{"type":"object"}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/responses" {
		t.Fatalf("path = %q, want /responses", gotPath)
	}
	if gotModel != "gpt-6-luna" || gotStore || gotToolType != "function:read" {
		t.Fatalf("model %q store %v tool %q", gotModel, gotStore, gotToolType)
	}
	if len(gotInput) != 1 || gotInput[0]["content"] != "Hello" {
		t.Fatalf("input = %#v", gotInput)
	}
	if resp.Text != "Hi" || resp.FinishReason != "tool_calls" {
		t.Fatalf("text %q reason %q", resp.Text, resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "read" || resp.ToolCalls[0].ID != "call_1" {
		t.Fatalf("calls %+v", resp.ToolCalls)
	}
	if !strings.Contains(string(resp.Replay), "rs_1") {
		t.Fatalf("replay %s", resp.Replay)
	}
	if resp.Usage.OutputTokens != 4 || resp.Usage.ReasoningTokens != 1 {
		t.Fatalf("usage %+v", resp.Usage)
	}

	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("kimi path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer chat.Close()
	kimi := NewClient(chat.URL, "sk")
	if _, err := kimi.Complete(context.Background(), provider.Request{
		ModelID:  "kimi-k2.7-code",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestResponsesReplaysToolTurn(t *testing.T) {
	var sawReplay, sawOutput bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []map[string]any `json:"input"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		for _, item := range body.Input {
			if item["id"] == "rs_1" {
				sawReplay = true
			}
			if item["type"] == "function_call_output" && item["call_id"] == "call_1" {
				sawOutput = true
			}
		}
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"done"}]}]}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sk")
	_, err := c.Complete(context.Background(), provider.Request{
		ModelID: "gpt-6-luna",
		Messages: []provider.Message{
			{Role: "user", Content: "read a.go"},
			{Role: "assistant", Replay: json.RawMessage(`[{"type":"reasoning","id":"rs_1"},{"type":"function_call","call_id":"call_1","name":"read","arguments":"{}"}]`)},
			{Role: "tool", ToolCallID: "call_1", Content: "file contents"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawReplay || !sawOutput {
		t.Fatalf("replay %v output %v", sawReplay, sawOutput)
	}
}
