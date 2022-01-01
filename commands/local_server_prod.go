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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
)

var localServerProdCmd = &console.Command{
	Category: "local",
	Name:     "server:prod",
	Aliases:  []*console.Alias{{Name: "server:prod"}},
	Usage:    "Switch a project to use Symfony's production environment",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "off", Usage: "Disable prod mode"},
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		beacon := filepath.Join(projectDir, ".prod")
		if c.Bool("off") {
			return errors.WithStack(os.Remove(beacon))
		}
		if f, err := os.OpenFile(beacon, os.O_RDONLY|os.O_CREATE, 0666); err != nil {
			return errors.WithStack(err)
		} else {
			f.Close()
		}
		return nil
	},
}
