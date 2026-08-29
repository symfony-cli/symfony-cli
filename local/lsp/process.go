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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
)

type NativeProcessRunner struct{}

func (NativeProcessRunner) Run(executable string, arguments []string, directory string, environment []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(executable, arguments...)
	cmd.Dir = directory
	cmd.Env = environment
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return ExitWrapperFailure, fmt.Errorf("unable to start Symfony Language Tools: %w", err)
	}

	finished := make(chan error, 1)
	go func() {
		finished <- cmd.Wait()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, forwardedSignals()...)
	defer signal.Stop(signals)
	for {
		select {
		case received := <-signals:
			_ = signalProcess(cmd, received)
		case err := <-finished:
			if err == nil {
				return 0, nil
			}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return processExitCode(exitError), nil
			}

			return ExitWrapperFailure, err
		}
	}
}
