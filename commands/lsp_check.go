/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
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
	"github.com/symfony-cli/symfony-cli/local/lsp"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var lspCheckCmd = newLspCheckCommand(func(arguments []string) (int, error) {
	return lsp.NewChecker(util.GetHomeDir()).Run(arguments)
})

func newLspCheckCommand(run func([]string) (int, error)) *console.Command {
	return &console.Command{
		Category:    "lsp",
		Name:        "check",
		Usage:       "Run Symfony Language Tools diagnostics",
		Description: "Runs the managed Symfony Language Tools checker with the project's PHP version and environment.",
		FlagParsing: console.FlagParsingSkipped,
		Args: []*console.Arg{
			{Name: "arguments", Optional: true, Slice: true},
		},
		Action: func(c *console.Context) error {
			exitCode, err := run(lspArguments(c))
			if err != nil {
				terminal.Eprintln(err)
				return console.Exit("", lsp.ExitWrapperFailure)
			}
			if exitCode != 0 {
				return console.Exit("", exitCode)
			}

			return nil
		},
	}
}

func lspArguments(c *console.Context) []string {
	for index, argument := range os.Args[1:] {
		if argument == c.Command.UserName {
			return os.Args[index+2:]
		}
	}

	return c.Args().Slice()
}

var lspCacheDirCmd = &console.Command{
	Category: "lsp",
	Name:     "cache-dir",
	Usage:    "Display the Symfony Language Tools cache directory",
	Action: func(c *console.Context) error {
		terminal.Println(lsp.ToolCacheDir(util.GetHomeDir()))
		return nil
	},
}
