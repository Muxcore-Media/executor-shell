//go:build !unix

package internal

import (
	"os/exec"
	"time"
)

// configureProcessGroup bounds how long Wait blocks on inherited pipes after
// cancellation on platforms without process groups.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.WaitDelay = 2 * time.Second
}
