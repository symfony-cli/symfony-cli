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
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

var localProxyStatusCmd = &console.Command{
	Category: "local",
	Name:     "proxy:status",
	Aliases:  []*console.Alias{{Name: "proxy:status"}},
	Usage:    "Get the local proxy server status",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		terminal.Println("<info>Local Proxy Server</>")

		pidFile := pid.New("__proxy__", nil)
		if !pidFile.IsRunning() {
			terminal.Println("    <error>Not Running</>")
			return nil
		}

		_, _ = terminal.Printfln("    Listening on <href=%s://127.0.0.1:%d>%s://127.0.0.1:%d</>", pidFile.Scheme, pidFile.Port, pidFile.Scheme, pidFile.Port)

		_, _ = terminal.Println()
		_, _ = terminal.Println("<info>Configured Web Servers</>")
		return printConfiguredServers()
	},
}
