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
	"fmt"

	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/terminal"
)

var doctrineCheckServerVersionSettingCmd = &console.Command{
	Name:   "doctrine:check-server-version-setting",
	Usage:  "Check if Doctrine server version is configured explicitly",
	Hidden: console.Hide,
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		logger := terminal.Logger.Output(zerolog.ConsoleWriter{Out: terminal.Stderr}).With().Timestamp().Logger()
		if err := checkDoctrineServerVersionSetting(projectDir, logger); err != nil {
			return err
		}

		fmt.Println("✅ Doctrine server version is set properly.")
		return nil
	},
}
