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

package book

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

func (b *Book) Clone(version string) error {
	ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
	ui.Section("Checking Book Requirements")
	ready, err := CheckRequirements()
	if err != nil {
		return err
	}
	terminal.Println("")
	if !ready {
		return errors.New("You should fix the reported issues before starting reading the book.")
	}

	ui.Section("Cloning the Repository")
	cmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/the-fast-track/book-%s", version), b.Dir)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error cloning the Git repository for the book")
	}
	terminal.Println("")

	os.Chdir(b.Dir)
	// checkout the first step by default
	ui.Section("Getting Ready for the First Step of the Book")
	if err := b.Checkout("3"); err != nil {
		terminal.Println("")
		if !b.Debug {
			terminal.Println("Re-run the command with <comment>--debug</> to get more information about the error")
			terminal.Println("")
		}
		return err
	}
	return nil
}
