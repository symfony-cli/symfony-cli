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
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/logs"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

var localServerLogCmd = &console.Command{
	Category: "local",
	Name:     "server:log",
	Aliases:  []*console.Alias{{Name: "server:log"}},
	Usage:    "Display local web server logs",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "no-follow", Aliases: []string{"no-tail"}, Usage: "Do no tail the logs"},
		&console.Int64Flag{Name: "lines", Aliases: []string{"n"}, DefaultValue: 0, Usage: "Number of lines to display at start"},
		&console.BoolFlag{Name: "no-humanize", Usage: "Do not format JSON logs"},
		&console.StringSliceFlag{
			Name:  "file",
			Usage: "Use this file for application logs",
		},
		&console.BoolFlag{Name: "no-app-logs", Usage: "Do not display the application logs"},
		&console.BoolFlag{Name: "no-worker-logs", Usage: "Do not display the worker logs"},
		&console.BoolFlag{Name: "no-server-logs", Usage: "Do not display web server/PHP logs"},
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		tailer := logs.Tailer{
			Follow:       !c.Bool("no-follow"),
			LinesNb:      c.Int64("lines"),
			AppLogs:      c.StringSlice("file"),
			NoHumanize:   c.Bool("no-humanize"),
			NoAppLogs:    c.Bool("no-app-logs"),
			NoWorkerLogs: c.Bool("no-worker-logs"),
			NoServerLogs: c.Bool("no-server-logs"),
		}

		if err := tailer.Watch(pid.New(projectDir, nil)); err != nil {
			return err
		}

		return tailer.Tail(terminal.Stderr)
	},
}
