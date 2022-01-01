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

package commands

import (
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
)

var (
	composerWrapper = &console.Command{
		Name:   "composer",
		Usage:  "Runs Composer without memory limit",
		Hidden: console.Hide,
		Action: func(c *console.Context) error {
			return console.IncorrectUsageError{ParentError: errors.New(`This command can only be run as "symfony composer"`)}
		},
	}
	binConsoleWrapper = &console.Command{
		Name:   "console",
		Usage:  "Runs the Symfony Console (bin/console) for current project",
		Hidden: console.Hide,
		Action: func(c *console.Context) error {
			return console.IncorrectUsageError{ParentError: errors.New(`This command can only be run as "symfony console"`)}
		},
	}
	phpWrapper = &console.Command{
		Usage:  "Runs the named binary using the configured PHP version",
		Hidden: console.Hide,
		Action: func(c *console.Context) error {
			return console.IncorrectUsageError{ParentError: errors.New(`This command can only be run as "symfony php*"`)}
		},
	}
)

func init() {
	for _, name := range php.GetBinaryNames() {
		phpWrapper.Aliases = append(phpWrapper.Aliases, &console.Alias{Name: name, Hidden: console.Hide()})
	}
}
