//go:build unix

package internal

import (
	"os/exec"
	"syscall"
	"time"
)

// configureProcessGroup runs the shell in its own process group so cancelling
// the request kills the shell and every child it spawned. Without this,
// exec.CommandContext only kills `sh`, and Wait blocks until grandchildren
// that still hold the output pipes exit on their own.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 2 * time.Second
}
