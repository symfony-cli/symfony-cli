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
	"errors"
	"fmt"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

var localProxyUrlCmd = &console.Command{
	Category: "local",
	Name:     "proxy:url",
	Aliases:  []*console.Alias{{Name: "proxy:url"}},
	Usage:    "Get the local proxy server URL",
	Description: `Get the local proxy server URL, for example if you need to define HTTP_PROXY/HTTPS_PROXY environment variables
when running an external program:

e.g. with Blackfire: 
	HTTP_PROXY=$(symfony proxy:url) HTTPS_PROXY=$(symfony proxy:url) blackfire curl ...
	
e.g. with Cypress: 
	HTTP_PROXY=$(symfony proxy:url) HTTPS_PROXY=$(symfony proxy:url) ./node_modules/bin/cypress ...
`,
	Action: func(c *console.Context) error {
		pidFile := pid.New("__proxy__", nil)
		if !pidFile.IsRunning() {
			return errors.New("The proxy server is not running")
		}

		url := fmt.Sprintf("%s://127.0.0.1:%d", pidFile.Scheme, pidFile.Port)
		terminal.Print(url)

		return nil
	},
}
