/*
 * Copyright (c) 2025-present Fabien Potencier <fabien@symfony.com>
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

package php

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/symfony-cli/util"
)

type PieResult struct {
	code  int
	error error
}

func (p PieResult) Error() string {
	if p.error != nil {
		return p.error.Error()
	}

	return ""
}

func (p PieResult) ExitCode() int {
	return p.code
}

func PieExecutor(dir string, args, env []string, stdout, stderr, logger io.Writer, debugLogger zerolog.Logger) (*Executor, error) {
	e := &Executor{
		Dir:        dir,
		BinName:    "php",
		Stdout:     stdout,
		Stderr:     stderr,
		SkipNbArgs: -1,
		ExtraEnv:   env,
		Logger:     debugLogger,
	}

	if piePath := os.Getenv("SYMFONY_PIE_PATH"); piePath != "" {
		debugLogger.Debug().Str("SYMFONY_PIE_PATH", piePath).Msg("SYMFONY_PIE_PATH has been defined. User is taking control over PIE detection and execution.")
		e.Args = append([]string{piePath}, args...)
	} else if path, err := e.findPie(); err == nil && isPHPScript(path) {
		e.Args = append([]string{"php", path}, args...)
	} else {
		reason := "No PIE installation found."
		if path != "" {
			reason = fmt.Sprintf("Detected PIE file (%s) is not a valid PHAR or PHP script.", path)
		}
		fmt.Fprintln(logger, "  WARNING:", reason)
		fmt.Fprintln(logger, "  Downloading PIE for you, but it is recommended to install PIE yourself, instructions available at https://github.com/php/pie")
		// we don't store it under bin/ to avoid it being found by findPie as we want to only use it as a fallback
		binDir := filepath.Join(util.GetHomeDir(), "pie")
		if path, err = downloadPie(binDir); err != nil {
			return nil, errors.Wrap(err, "unable to find pie, get it at https://github.com/php/pie")
		}
		e.Args = append([]string{"php", path}, args...)
		fmt.Fprintf(logger, "  (running %s)\n\n", e.CommandLine())
	}

	return e, nil
}

func Pie(dir string, args, env []string, stdout, stderr, logger io.Writer, debugLogger zerolog.Logger) PieResult {
	e, err := PieExecutor(dir, args, env, stdout, stderr, logger, debugLogger)
	if err != nil {
		return PieResult{
			code:  1,
			error: errors.WithStack(err),
		}
	}

	ret := e.Execute(false)
	if ret != 0 {
		return PieResult{
			code:  ret,
			error: errors.Errorf("unable to run %s", e.CommandLine()),
		}
	}
	return PieResult{}
}

func findPie(logger zerolog.Logger) (string, error) {
	for _, file := range []string{"pie", "pie.phar"} {
		logger.Debug().Str("source", "PIE").Msgf(`Looking for PIE in the PATH as "%s"`, file)
		if pharPath, _ := LookPath(file); pharPath != "" {
			logger.Debug().Str("source", "PIE").Msgf(`Found potential PIE as "%s"`, pharPath)
			return pharPath, nil
		}
	}

	return "", os.ErrNotExist
}

func downloadPie(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "pie.phar")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	piePhar, err := downloadPiePhar()
	if err != nil {
		return "", err
	}

	err = os.WriteFile(path, piePhar, 0755)
	if err != nil {
		return "", err
	}

	return path, nil
}

func downloadPiePhar() ([]byte, error) {
	resp, err := http.Get("https://github.com/php/pie/releases/latest/download/pie.phar")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
