package reexec

import (
	"os"
	"syscall"
)

func setsid(s *syscall.SysProcAttr) {
}

func shouldSignalBeIgnored(sig os.Signal) bool {
	return false
}
