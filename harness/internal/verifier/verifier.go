package verifier

import (
	"context"
	"strings"
	"time"

	"github.com/PLACEHOLDER/muxon/harness/internal/subproc"
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
		return Result{}, errEmptyVerify
	}
	start := time.Now()
	out, exitCode, err := subproc.Run(ctx, workspaceRoot, argv, nil)
	dur := time.Since(start).Milliseconds()
	if err != nil && exitCode != 0 && exitCode != 1 {
		return Result{}, err
	}
	if exitCode < 0 {
		return Result{}, err
	}
	excerpt := excerptOutput(string(out), 20)
	return Result{
		Command:    argv,
		ExitCode:   exitCode,
		Passed:     exitCode == 0,
		DurationMS: dur,
		Excerpt:    excerpt,
	}, nil
}

var errEmptyVerify = &verifyError{"empty verify command"}

type verifyError struct{ msg string }

func (e *verifyError) Error() string { return e.msg }

func excerptOutput(s string, maxLines int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}
