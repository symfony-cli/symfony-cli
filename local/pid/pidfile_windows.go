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

package pid

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

func kill(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return errors.WithStack(err)
	}

	// TerminateProcess fails on a process that already exited (for example
	// with "Access is denied"), don't report an error in this case
	if err := p.Kill(); err != nil && isRunning(pid) {
		return errors.WithStack(err)
	}

	return nil
}

// Windows keeps the process object (and thus its PID) alive as long as a
// handle to it is open, so signal-based liveness checks report just-exited
// processes as still running. Waiting on the process handle with a zero
// timeout tells whether the process actually terminated.
func isRunning(pid int) bool {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	event, err := windows.WaitForSingleObject(handle, 0)
	return err == nil && event == uint32(windows.WAIT_TIMEOUT)
}
