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
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localProxyDetachDomainCmd = &console.Command{
	Category: "local",
	Name:     "proxy:domain:detach",
	Aliases:  []*console.Alias{{Name: "proxy:domain:detach"}, {Name: "proxy:domain:remove", Hidden: true}},
	Usage:    "Detach domains from the proxy",
	Args: []*console.Arg{
		{Name: "domains", Optional: true, Description: "The domains to detach", Slice: true},
	},
	Action: func(c *console.Context) error {
		homeDir := util.GetHomeDir()
		config, err := proxy.Load(homeDir)
		if err != nil {
			return err
		}
		domains := c.Args().Tail()
		if err := config.RemoveDirDomains(domains); err != nil {
			return err
		}
		terminal.Println("<info>The following domains are not defined anymore on the proxy:</>")
		for _, domain := range domains {
			terminal.Printfln(" * http://%s", domain)
		}
		return nil
	},
}
