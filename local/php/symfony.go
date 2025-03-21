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

package php

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/symfony-cli/envs"
)

// SymfonyConsoleExecutor returns an Executor prepared to run Symfony Console.
// It returns an error if no console binary is found.
func SymfonyConsoleExecutor(logger zerolog.Logger, args []string) (*Executor, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for {
		consolePaths := []string{"bin/console", "app/console"}
		if consolePath, isConsolePathSpecified := envs.LookupEnv(dir, "SYMFONY_CONSOLE_PATH"); isConsolePathSpecified {
			consolePaths = []string{consolePath}
		}

		for _, consolePath := range consolePaths {
			logger.Debug().Str("consolePath", consolePath).Str("directory", dir).Msgf("Looking for Symfony console")
			consolePath = filepath.Join(dir, consolePath)
			if _, err := os.Stat(consolePath); err == nil {
				return &Executor{
					BinName: "php",
					Logger:  logger,
					Args:    append([]string{"php", consolePath}, args...),
				}, nil
			}
		}

		upDir := filepath.Dir(dir)
		if upDir == dir || upDir == "." {
			break
		}
		dir = upDir
	}

	return nil, errors.New("No console binary found")
}
