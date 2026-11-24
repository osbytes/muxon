package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
)

func TestModelsList(t *testing.T) {
	t.Setenv("MUXON_HOME", t.TempDir())
	var buf bytes.Buffer
	modelsListCmd.SetOut(&buf)
	modelsListCmd.SetErr(&buf)
	if err := modelsListCmd.RunE(modelsListCmd, nil); err != nil {
		t.Fatal(err)
	}
	m, err := config.LoadMergedManifest()
	if err != nil {
		t.Fatal(err)
	}
	w, err := m.ResolveRole("worker")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), w.ID) {
		t.Fatalf("%s", buf.String())
	}
}
