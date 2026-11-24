package config

import (
	"testing"
)

func TestResolveAPIKeyMissing(t *testing.T) {
	t.Setenv("MUXON_TEST_KEY", "")
	_, err := ResolveAPIKey("MUXON_TEST_KEY")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveAPIKeyPresent(t *testing.T) {
	t.Setenv("MUXON_TEST_KEY", "secret")
	v, err := ResolveAPIKey("MUXON_TEST_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if v != "secret" {
		t.Fatalf("got %q", v)
	}
}
