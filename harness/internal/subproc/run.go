package subproc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Run executes argv directly with env scrubbing and process-group timeout.
func Run(ctx context.Context, dir string, argv []string, scrubKeys []string) (stdout []byte, exitCode int, err error) {
	if err := validateArgv(argv); err != nil {
		return nil, 0, err
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = scrubEnv(os.Environ(), scrubKeys)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()
	code := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if ctx.Err() != nil {
			killProcessGroup(cmd.Process)
			return buf.Bytes(), -1, ctx.Err()
		} else {
			return nil, 0, runErr
		}
	}
	return buf.Bytes(), code, nil
}

func validateArgv(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty argv")
	}
	base := strings.ToLower(filepathBase(argv[0]))
	shells := map[string]bool{"sh": true, "bash": true, "zsh": true, "cmd": true, "powershell": true}
	if shells[base] {
		for _, a := range argv[1:] {
			if a == "-c" {
				return fmt.Errorf("shell -c invocations are denied")
			}
		}
	}
	return nil
}

func filepathBase(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func scrubEnv(env []string, extraKeys []string) []string {
	keySet := make(map[string]bool)
	for _, k := range extraKeys {
		if k != "" {
			keySet[k] = true
		}
	}
	out := make([]string, 0, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 1 {
			continue
		}
		key := parts[0]
		if keySet[key] {
			continue
		}
		upper := strings.ToUpper(key)
		if strings.HasSuffix(upper, "_API_KEY") || strings.HasSuffix(upper, "_TOKEN") || strings.HasSuffix(upper, "_SECRET") || strings.HasPrefix(upper, "AWS_") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func killProcessGroup(p *os.Process) {
	if p == nil {
		return
	}
	_ = syscall.Kill(-p.Pid, syscall.SIGKILL)
}

// RunWithTimeout runs argv with a deadline.
func RunWithTimeout(dir string, argv []string, scrubKeys []string, timeout time.Duration) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Run(ctx, dir, argv, scrubKeys)
}
