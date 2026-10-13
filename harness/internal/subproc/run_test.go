package subproc

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDenyBashC(t *testing.T) {
	err := validateArgv([]string{"bash", "-c", "echo hi"})
	if err == nil {
		t.Fatal("expected deny")
	}
}

func TestTimeoutKillsGroup(t *testing.T) {
	if os.Getenv("RUN_SUBPROC_TIMEOUT") != "1" {
		t.Skip("set RUN_SUBPROC_TIMEOUT=1 to run")
	}
	script := filepath.Join(t.TempDir(), "slow.sh")
	body := "#!/bin/sh\nsleep 30 &\nwait\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, _, err := RunWithTimeout(t.TempDir(), []string{script}, nil, 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout took too long")
	}
}
