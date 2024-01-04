//go:build !windows
// +build !windows

package local

import (
	"os/exec"
	"syscall"
)

func buildCmd(cmd *exec.Cmd, foreground bool) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// isolate each command in a new process group that we can cleanly send
		// signal to when we want to stop it
		Setpgid:    true,
		Foreground: foreground,
	}

	return nil
}
