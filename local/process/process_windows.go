package process

import (
	"os/exec"
	"strconv"
	"syscall"
)

func deathsig(sysProcAttr *syscall.SysProcAttr) {
}

func kill(cmd *exec.Cmd) error {
	c := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	if err := c.Run(); err == nil {
		return nil
	}
	return cmd.Process.Signal(syscall.SIGKILL)
}
