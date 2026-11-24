package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const dotEnvName = ".env"

var utf8BOM = string([]byte{0xEF, 0xBB, 0xBF})

// FindDotEnv returns the nearest .env file at or above dir, stopping at the
// first directory that holds .git. It returns "" when there is none.
func FindDotEnv(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, dotEnvName)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// LoadDotEnv sets environment variables from the nearest .env file at or above
// dir. Variables already set in the environment win. It returns the file it
// read, or "" when there was none.
func LoadDotEnv(dir string) (string, error) {
	path, err := FindDotEnv(dir)
	if err != nil || path == "" {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	vars, err := ParseDotEnv(path, bufio.NewScanner(bytes.NewReader(b)))
	if err != nil {
		return "", err
	}
	for k, v := range vars {
		if _, set := os.LookupEnv(k); set {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return "", fmt.Errorf("%s: set %s: %w", path, k, err)
		}
	}
	return path, nil
}

// ParseDotEnv reads KEY=VALUE lines. Blank lines, # comments, an optional
// export prefix, and single or double quotes around values are accepted.
func ParseDotEnv(name string, sc *bufio.Scanner) (map[string]string, error) {
	vars := map[string]string{}
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if n == 1 {
			line = strings.TrimPrefix(line, utf8BOM)
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("%s:%d: want KEY=VALUE", name, n)
		}
		vars[key] = dotEnvValue(strings.TrimSpace(val))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return vars, nil
}

func dotEnvValue(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}
