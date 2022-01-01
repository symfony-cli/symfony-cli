//go:build !windows
// +build !windows

/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

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
