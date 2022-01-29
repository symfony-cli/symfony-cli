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
	"github.com/symfony-cli/symfony-cli/book"
	"github.com/symfony-cli/terminal"
)

var bookCheckReqsCmd = &console.Command{
	Category: "book",
	Name:     "check-requirements",
	Usage:    `Check that you have all the pre-requisites locally to code while reading the "Symfony: The Fast Track" book`,
	Aliases:  []*console.Alias{{Name: "book:check"}},
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)

		ready, err := book.CheckRequirements()
		if err != nil {
			return err
		}
		terminal.Println("")
		if ready {
			ui.Success("Congrats! You are ready to start reading the book.")
			return nil
		}
		return console.Exit("You should fix the reported issues before starting reading the book.", 1)
	},
}
