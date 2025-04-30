/*
 * Copyright (c) 2025-present Fabien Potencier <fabien@symfony.com>
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
	"github.com/symfony-cli/symfony-cli/local/mcp"
)

var localMcpServerStartCmd = &console.Command{
	Category:    "local",
	Name:        "mcp:start",
	Aliases:     []*console.Alias{{Name: "mcp:start"}},
	Usage:       "Run a local MCP server",
	Description: localWebServerProdWarningMsg,
	Args: console.ArgDefinition{
		{Name: "projectDir", Optional: true, Description: "The path to the Symfony application"},
	},
	Action: func(c *console.Context) error {
		server, err := mcp.NewServer(c.Args().Get("projectDir"))
		if err != nil {
			return err
		}
		return server.Start()
	},
}
