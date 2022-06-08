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

package process

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func createSysProcAttr() *syscall.SysProcAttr {
	return &unix.SysProcAttr{
		// the following helps with killing the main process and its children
		// see https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
		Setpgid:   true,
		Pdeathsig: unix.SIGKILL,
	}
}

func kill(cmd *exec.Cmd) error {
	return unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
}
