package config

import "testing"

func TestSaveLastModelSurvivesProviderSave(t *testing.T) {
	t.Setenv("THEE_HOME", t.TempDir())
	if err := SaveLastProvider("opencode-go"); err != nil {
		t.Fatal(err)
	}
	if err := SaveLastModel("gpt-6-luna"); err != nil {
		t.Fatal(err)
	}
	if err := SaveLastProvider("opencode-go"); err != nil {
		t.Fatal(err)
	}
	if LastModel() != "gpt-6-luna" || LastProvider() != "opencode-go" {
		t.Fatalf("provider %q model %q", LastProvider(), LastModel())
	}
	if err := SaveLastProvider("deepinfra"); err != nil {
		t.Fatal(err)
	}
	if LastModel() != "" {
		t.Fatalf("LastModel() = %q after provider switch, want empty", LastModel())
	}
}
