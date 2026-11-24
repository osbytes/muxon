//go:build unix

package subproc

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcessGroup starts cmd in its own process group so the whole group can be killed.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree sends SIGKILL to the process group led by p, falling back to p alone.
func killProcessTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	if err := syscall.Kill(-p.Pid, syscall.SIGKILL); err != nil {
		return p.Kill()
	}
	return nil
}
