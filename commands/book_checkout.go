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
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/book"
	"github.com/symfony-cli/terminal"
)

var bookCheckoutCmd = &console.Command{
	Category: "book",
	Name:     "checkout",
	Usage:    `Check out a step of the "Symfony: The Fast Track" book repository`,
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "debug", Usage: "Display commands output"},
		&console.BoolFlag{Name: "force", Usage: "Force the use of the command without checking pre-requisites"},
	},
	Args: []*console.Arg{
		{Name: "step", Description: "The step of the book to checkout (code at the end of the step)"},
	},
	Action: func(c *console.Context) error {
		dir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return errors.WithStack(err)
		}

		book := &book.Book{
			Dir:   dir,
			Debug: c.Bool("debug"),
			Force: c.Bool("force"),
		}
		if !c.Bool("force") {
			if err := book.CheckRepository(); err != nil {
				return errors.WithStack(err)
			}
		}
		if err := book.Checkout(c.Args().Get("step")); err != nil {
			terminal.Println("")
			if !c.Bool("debug") {
				terminal.Println("Re-run the command with <comment>--debug</> to get more information about the error")
				terminal.Println("")
			}
			return errors.WithStack(err)
		}
		return nil
	},
}
