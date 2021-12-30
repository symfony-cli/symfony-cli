//go:build !linux && !windows
// +build !linux,!windows

package process

import (
	"os/exec"
	"syscall"
)

func deathsig(sysProcAttr *syscall.SysProcAttr) {
	// the following helps with killing the main process and its children
	// see https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	sysProcAttr.Setpgid = true
}

func kill(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
