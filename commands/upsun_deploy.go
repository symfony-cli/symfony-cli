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
	"os"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/upsun"
	"github.com/symfony-cli/terminal"
)

var upsunDeployCmd = &console.Command{
	Category: "cloud",
	Name:     "deploy",
	Aliases:  []*console.Alias{{Name: "upsun:deploy", Hidden: true}, {Name: "deploy", Hidden: true}},
	Hidden:   console.Hide,
	Flags: func() []console.Flag {
		for _, cmd := range upsun.Commands {
			if cmd.Category == "cloud:environment" && cmd.Name == "push" {
				return cmd.Flags
			}
		}
		return []console.Flag{}
	}(),
	Action: func(c *console.Context) error {
		terminal.Printf(`<warning> /////////////// WARNING \\\\\\\\\\\\\\\ </>
The <info>%s</> command is ambiguous.

You should use <info>env:push</> or <info>env:deploy</> instead.

Historically, the <comment>%s</> command was an alias for <comment>env:push</> on Symfony Cloud.
But Upsun (formerly Platform.sh) now has an <comment>env:deploy</> command.

To deploy your application like before, use <comment>env:push</>.
Or <href=https://docs.upsun.com/administration/cli/reference.html#environmentdeploy>read</> about the new <comment>env:deploy</> command.
<warning> /////////////// WARNING \\\\\\\\\\\\\\\ </>

`, c.Command.FullName(), c.Command.FullName(),
		)
		os.Args[1] = "cloud:environment:push"
		return c.App.Run(os.Args)
	},
}
