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

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/terminal"
)

var localVariableExposeFromTunnelCmd = &console.Command{
	Category: "local",
	Name:     "var:expose-from-tunnel",
	Aliases:  []*console.Alias{{Name: "var:expose-from-tunnel"}},
	Usage:    "Expose tunnel service environment variables locally",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "off", Usage: "Stop exposing tunnel service environment variables"},
	},
	Action: func(c *console.Context) error {
		dir := c.String("dir")
		if dir == "" {
			var err error
			if dir, err = os.Getwd(); err != nil {
				return errors.WithStack(err)
			}
		}

		project, err := platformsh.ProjectFromDir(dir, false)
		if err != nil {
			return errors.WithStack(err)
		}
		tunnel := envs.Tunnel{Project: project}

		if c.Bool("off") {
			terminal.Eprintln("Stop exposing tunnel service environment variables")
			return tunnel.Expose(false)
		}

		terminal.Eprintln("Exposing tunnel service environment variables")
		return tunnel.Expose(true)
	},
}
