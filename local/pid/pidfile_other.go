//go:build !windows
// +build !windows

package pid

import "syscall"

func kill(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}
	return syscall.Kill(-pgid, syscall.SIGTERM)
}
