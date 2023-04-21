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
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/symfony-cli/cert"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localServerCAUninstallCmd = &console.Command{
	Category: "local",
	Name:     "server:ca:uninstall",
	Aliases:  []*console.Alias{{Name: "server:ca:uninstall"}},
	Usage:    "Uninstall the local Certificate Authority",
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		certsDir := filepath.Join(util.GetHomeDir(), "certs")
		ca, err := cert.NewCA(certsDir)
		if err != nil {
			return nil
		}
		if !ca.HasCA() {
			ui.Success("The local Certificate Authority is not installed yet")
			return nil
		}
		if err = ca.LoadCA(); err != nil {
			return errors.Wrap(err, "failed to load the local Certificate Authority")
		}
		_ = ca.Uninstall()
		_ = os.RemoveAll(certsDir)
		ui.Success("The local Certificate Authority has been uninstalled")
		return nil
	},
}
