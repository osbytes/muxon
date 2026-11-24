package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseDotEnv(t *testing.T) {
	in := utf8BOM + "# keys\n\nexport A=1\nB = \"two words\"\nC='x # y'\nD=plain # note\nE=\n"
	got, err := ParseDotEnv("test.env", bufio.NewScanner(strings.NewReader(in)))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"A": "1", "B": "two words", "C": "x # y", "D": "plain", "E": ""}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ParseDotEnv (-want +got):\n%s", diff)
	}
}

func TestParseDotEnvBadLine(t *testing.T) {
	for _, in := range []string{"NOEQUALS", "=v", "BAD KEY=v"} {
		if _, err := ParseDotEnv("test.env", bufio.NewScanner(strings.NewReader(in))); err == nil {
			t.Errorf("ParseDotEnv(%q): got nil error", in)
		}
	}
}

func unsetForTest(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDotEnvWalksUpToRepoRoot(t *testing.T) {
	unsetForTest(t, "THEE_DOTENV_A", "THEE_DOTENV_B")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(root, ".env"), "THEE_DOTENV_A=from-file\nTHEE_DOTENV_B=from-file\n")
	t.Setenv("THEE_DOTENV_B", "from-env")
	sub := filepath.Join(root, "harness", "cmd")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	path, err := LoadDotEnv(sub)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, ".env") {
		t.Errorf("LoadDotEnv: got path %q", path)
	}
	if got := os.Getenv("THEE_DOTENV_A"); got != "from-file" {
		t.Errorf("THEE_DOTENV_A: got %q, want from-file", got)
	}
	if got := os.Getenv("THEE_DOTENV_B"); got != "from-env" {
		t.Errorf("THEE_DOTENV_B: got %q, want from-env", got)
	}
}

func TestFindDotEnvStopsAtRepoRoot(t *testing.T) {
	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, ".env"), "THEE_DOTENV_OUTER=1\n")
	repo := filepath.Join(outer, "repo")
	writeFile(t, filepath.Join(repo, ".git"), "gitdir: elsewhere\n")
	path, err := FindDotEnv(repo)
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Errorf("FindDotEnv: got %q, want none", path)
	}
}
