package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Result is a verify run outcome.
type Result struct {
	Command    []string
	ExitCode   int
	Passed     bool
	DurationMS int64
	Excerpt    string
}

// Run executes argv directly (no shell).
func Run(ctx context.Context, workspaceRoot string, argv []string) (Result, error) {
	if len(argv) == 0 {
		return Result{}, fmt.Errorf("empty verify command")
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = workspaceRoot
	cmd.Env = os.Environ()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	dur := time.Since(start).Milliseconds()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return Result{}, err
		}
	}
	excerpt := excerptOutput(buf.String(), 20)
	return Result{
		Command:    argv,
		ExitCode:   exitCode,
		Passed:     exitCode == 0,
		DurationMS: dur,
		Excerpt:    excerpt,
	}, nil
}

func excerptOutput(s string, maxLines int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}
