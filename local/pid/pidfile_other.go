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

package pid

import (
	"errors"
	"syscall"
	"time"
)

func kill(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}

	// Send SIGTERM to the process group
	err = syscall.Kill(-pgid, syscall.SIGINT)
	if err != nil {
		return err
	}

	// Wait for the process group to exit gracefully with a timeout of 5 seconds
	done := make(chan error, 1)
	go func() {
		_, err := syscall.Wait4(-pgid, nil, 0, nil)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-time.After(5 * time.Second):
		return errors.New("timeout waiting for process group to exit gracefully")
	}

	// Send SIGKILL to the process group
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
