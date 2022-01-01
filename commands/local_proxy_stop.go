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

var localProxyStopCmd = &console.Command{
	Category: "local",
	Name:     "proxy:stop",
	Aliases:  []*console.Alias{{Name: "proxy:stop"}},
	Usage:    "Stop the local proxy server",
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		p := pid.New("__proxy__", nil)
		if !p.IsRunning() {
			ui.Success("The proxy server is not running")
			return nil
		}
		if err := p.Stop(); err != nil {
			return err
		}
		ui.Success("Stopped the proxy server successfully")
		return nil
	},
}
