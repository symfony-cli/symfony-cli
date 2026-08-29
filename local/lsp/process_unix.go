//go:build !windows

/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
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

package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func forwardedSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM}
}

func signalProcess(cmd *exec.Cmd, signal os.Signal) error {
	if err := cmd.Process.Signal(signal); err != nil {
		return fmt.Errorf("unable to signal Symfony Language Tools: %w", err)
	}

	return nil
}

func processExitCode(exitError *exec.ExitError) int {
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	if code := exitError.ExitCode(); code >= 0 {
		return code
	}

	return ExitWrapperFailure
}
