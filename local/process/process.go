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
	"bufio"
	"context"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Process struct {
	Dir           string
	Path          string
	Args          []string
	Logger        zerolog.Logger
	ProcessLogger func(string)
	Env           []string
}

func (p *Process) Run(ctx context.Context) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, p.Path, p.Args...)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if p.ProcessLogger == nil {
		p.ProcessLogger = func(t string) {
			p.Logger.Info().Msg(t)
		}
	}
	errReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	go func() {
		s := bufio.NewScanner(outReader)
		for s.Scan() {
			p.ProcessLogger(s.Text())
		}
	}()
	go func() {
		s := bufio.NewScanner(errReader)
		for s.Scan() {
			p.ProcessLogger(s.Text())
		}
	}()

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, p.Env...)
	cmd.SysProcAttr = createSysProcAttr()
	if err := cmd.Start(); err != nil {
		return nil, errors.WithStack(err)
	}
	go func() {
		p.Logger.Debug().Msg("started")
		<-ctx.Done()
		kill(cmd)
		p.Logger.Debug().Msg("stopped")
	}()
	return cmd, nil
}
