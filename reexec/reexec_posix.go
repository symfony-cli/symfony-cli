//go:build !windows
// +build !windows

package reexec

import (
	"os"
	"syscall"
)

func setsid(s *syscall.SysProcAttr) {
	// Setsid is used to detach the process from the parent (normally a shell)
	//
	// The disowning of a child process is accomplished by executing the system call
	// setpgrp() or setsid(), (both of which have the same functionality) as soon as
	// the child is forked. These calls create a new process session group, make the
	// child process the session leader, and set the process group ID to the process
	// ID of the child. https://bsdmag.org/unix-kernel-system-calls/
	s.Setsid = true
}

func shouldSignalBeIgnored(sig os.Signal) bool {
	// this one in particular should be skipped as we don't want to
	// send it back to child because it's about it
	return sig == syscall.SIGCHLD
}
