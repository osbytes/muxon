package config

import (
	"strings"
	"testing"
)

func TestDoctorReportsMissingKeyNotValue(t *testing.T) {
	t.Setenv("DOCTOR_TEST_KEY", "")
	m, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Providers["deepinfra"] = ProviderDef{
		BaseURL:   "https://example.com",
		APIKeyEnv: "DOCTOR_TEST_KEY",
	}
	rep := RunDoctor(m)
	text := strings.Join(rep.Lines, "\n")
	if !strings.Contains(text, "DOCTOR_TEST_KEY") {
		t.Fatalf("%s", text)
	}
	if strings.Contains(text, "secret") {
		t.Fatalf("leaked secret in %s", text)
	}
}
