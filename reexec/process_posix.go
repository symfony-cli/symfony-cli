//go:build !windows
// +build !windows

package reexec

import (
	"os"
	"syscall"
	"time"
)

func Getppid() int {
	return os.Getppid()
}

func waitForProcess(ps *os.Process) {
	for {
		time.Sleep(1 * time.Second)
		err := ps.Signal(syscall.Signal(0))
		if err != nil {
			return
		}
	}
}
