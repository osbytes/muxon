package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gcAge = 14 * 24 * time.Hour

// GC removes session directories older than 14 days and shadow refs under refs/muxon/sessions.
func GC(workspace string, now time.Time) (int, error) {
	n, err := gcSessions(workspace, now, gcAge)
	if err != nil {
		return n, err
	}
	refs, err := gcShadowRefs(workspace)
	return n + refs, err
}

func gcSessions(workspace string, now time.Time, olderThan time.Duration) (int, error) {
	dir := filepath.Join(workspace, ".muxon", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return removed, err
		}
		if now.Sub(info.ModTime()) < olderThan {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func gcShadowRefs(workspace string) (int, error) {
	if _, err := os.Stat(filepath.Join(workspace, ".git")); err != nil {
		return 0, nil
	}
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname)", "refs/muxon/sessions")
	cmd.Dir = workspace
	out, err := cmd.Output()
	if err != nil {
		return 0, nil
	}
	removed := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		ref := strings.TrimSpace(line)
		if ref == "" || !strings.HasPrefix(ref, "refs/muxon/sessions/") {
			continue
		}
		del := exec.Command("git", "update-ref", "-d", ref)
		del.Dir = workspace
		if err := del.Run(); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
