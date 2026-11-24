//go:build windows

package subproc

import (
	"os"
	"testing"
	"time"
)

func TestTimeoutKillsTreeWindows(t *testing.T) {
	if os.Getenv("RUN_SUBPROC_TIMEOUT") != "1" {
		t.Skip("set RUN_SUBPROC_TIMEOUT=1 to run")
	}
	start := time.Now()
	_, code, err := RunWithTimeout(t.TempDir(), []string{"cmd", "/C", "ping -n 30 127.0.0.1"}, nil, 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if code != -1 {
		t.Fatalf("exit code = %d, want -1", code)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout took too long")
	}
}
