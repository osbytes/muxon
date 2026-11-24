//go:build windows

package subproc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// setProcessGroup starts cmd in a new console process group, detached from Ctrl+C sent to the harness.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// killProcessTree force-kills p and its descendants with taskkill, falling back to p alone.
func killProcessTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	taskkill := "taskkill"
	if root := os.Getenv("SystemRoot"); root != "" {
		taskkill = filepath.Join(root, "System32", "taskkill.exe")
	}
	if err := exec.Command(taskkill, "/T", "/F", "/PID", strconv.Itoa(p.Pid)).Run(); err != nil { //nolint:gosec // G204: system taskkill.exe with a numeric PID
		return p.Kill()
	}
	return nil
}
