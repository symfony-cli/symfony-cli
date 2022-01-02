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
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localPhpListCmd = &console.Command{
	Category: "local",
	Name:     "php:list",
	Usage:    "List locally available PHP versions",
	Action: func(c *console.Context) error {
		s := terminal.NewSpinner(terminal.Stdout)
		s.Start()
		defer s.Stop()

		wd, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "unable to determine current dir")
		}
		homeDir := util.GetHomeDir()
		phpStore := phpstore.New(homeDir, true, terminal.Logger.Debug().Msgf)

		currentPHPPath := ""
		v, source, warning, _ := phpStore.BestVersionForDir(wd)
		if warning != "" {
			terminal.Eprintfln("<warning>WARNING</> %s", warning)
		}
		if v != nil {
			currentPHPPath = v.PHPPath
		}

		table := tablewriter.NewWriter(terminal.Stdout)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{terminal.Format("<header>Version</>"), terminal.Format("<header>Directory</>"), terminal.Format("<header>PHP CLI</>"), terminal.Format("<header>PHP FPM</>"), terminal.Format("<header>PHP CGI</>"), terminal.Format("<header>Server</>"), terminal.Format("<header>System?</>")})

		sep := string(os.PathSeparator)
		for _, v := range phpStore.Versions() {
			system := ""
			if v.IsSystem {
				system = "*"
			}
			phpPath := strings.Replace(v.PHPPath, v.Path+sep, "", 1)
			fpmPath := strings.Replace(v.FPMPath, v.Path+sep, "", 1)
			cgiPath := strings.Replace(v.CGIPath, v.Path+sep, "", 1)
			version := v.Version
			if v.PHPPath == currentPHPPath {
				version = terminal.Format("<options=reverse>" + version + "</>")
			}
			table.Append([]string{version, v.Path, phpPath, fpmPath, cgiPath, v.ServerTypeName(), system})
		}
		table.Render()

		terminal.Println("")

		if source != "" {
			terminal.Printf("The current PHP version is selected from %s\n", source)
		}

		terminal.Println("")
		terminal.Println("To control the version used in a directory, create a <comment>.php-version</> file that contains the version number (e.g. 7.2 or 7.2.15).")
		terminal.Println("If you're using Platform.sh, the version can also be specified in the <comment>.platform.app.yaml</> file.")

		return nil
	},
}
