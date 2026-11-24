package eval

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func copyTree(src, dst string) ([]string, error) {
	var written []string
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("check file %s already exists in the workspace", rel)
		}
		if err := copyFile(path, target, info.Mode()); err != nil {
			return err
		}
		written = append(written, target)
		return nil
	})
	return written, err
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func removePaths(paths []string) {
	for i := len(paths) - 1; i >= 0; i-- {
		_ = os.Remove(paths[i])
	}
}

func gitBaseline(dir string) {
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "eval@thee.local"},
		{"git", "config", "user.name", "thee eval"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", "baseline", "--allow-empty"},
	}
	for _, argv := range cmds {
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return
		}
	}
}

func findDir(name string, bases []string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(name) {
		st, err := os.Stat(name)
		if err != nil {
			return "", err
		}
		if !st.IsDir() {
			return "", fmt.Errorf("%s is not a directory", name)
		}
		return name, nil
	}
	for _, base := range bases {
		if base == "" {
			continue
		}
		cand := filepath.Join(base, name)
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("directory %q not found", name)
}

func fixtureCandidates(name string, extra []string) []string {
	var bases []string
	bases = append(bases, extra...)
	wd, _ := os.Getwd()
	bases = append(bases,
		wd,
		filepath.Join(wd, "testdata", "fixtures"),
		filepath.Join(wd, "harness", "testdata", "fixtures"),
		filepath.Join(wd, "..", "testdata", "fixtures"),
		filepath.Join(wd, "..", "..", "testdata", "fixtures"),
	)
	return bases
}

func overrideKey(ov map[string]string) string {
	if len(ov) == 0 {
		return ""
	}
	keys := make([]string, 0, len(ov))
	for k := range ov {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+ov[k])
	}
	return strings.Join(parts, ",")
}
