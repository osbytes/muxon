package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadEmbeddedManifestWorkerRole(t *testing.T) {
	m, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	model, err := m.ResolveRole("worker")
	if err != nil {
		t.Fatal(err)
	}
	if model.ID != "m0-worker" {
		t.Fatalf("model id: %q", model.ID)
	}
	if model.Capabilities.ToolProtocol != "native" {
		t.Fatalf("protocol: %q", model.Capabilities.ToolProtocol)
	}
}

func TestApplyDefaults(t *testing.T) {
	c := Capabilities{}
	ApplyDefaults(&c)
	want := Capabilities{
		ContextWindow: 32768,
		ToolProtocol:  "two_step",
		EditFormat:    "search_replace",
		PromptVariant: "compact",
	}
	if diff := cmp.Diff(want, c); diff != "" {
		t.Fatalf("ApplyDefaults (-want +got):\n%s", diff)
	}
}
