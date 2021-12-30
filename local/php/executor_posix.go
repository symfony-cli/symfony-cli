//+build !windows

package php

import (
	"os"
	"syscall"
)

func shouldSignalBeIgnored(sig os.Signal) bool {
	// this one in particular should be skipped as we don't want to
	// send it back to child because it's about it
	return sig == syscall.SIGCHLD
}

func symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
