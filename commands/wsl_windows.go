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
	"os"

	"github.com/symfony-cli/terminal"
)

func checkWSL() {
	if fi, err := os.Stat("/proc/version"); fi == nil || err != nil {
		return
	}

	ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
	ui.Error("Wrong binary for WSL")
	terminal.Println(`You are trying to run the Windows version of the Symfony CLI on WSL (Linux).
You must use the Linux version to use the Symfony CLI on WSL.

Download it at <href=https://symfony.com/download>https://symfony.com/download</>
`)
	os.Exit(1)
}
