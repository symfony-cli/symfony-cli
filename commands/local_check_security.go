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
	"fmt"
	"os"

	"github.com/fabpot/local-php-security-checker/v2/security"
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/terminal"
)

var localSecurityCheckCmd = &console.Command{
	Category: "local",
	Name:     "check:security",
	Aliases:  []*console.Alias{{Name: "security:check"}, {Name: "check:security"}, {Name: "local:security:check"}},
	Usage:    "Check security issues in project dependencies",
	Description: `Checks security issues in project dependencies. Without arguments, it looks
for a "composer.lock" file in the current directory. Pass it explicitly to check
a specific "composer.lock" file.`,
	Flags: []console.Flag{
		dirFlag,
		&console.StringFlag{
			Name:         "format",
			DefaultValue: "ansi",
			Usage:        "The output format (ansi, text, markdown, json, junit, or yaml)",
			Validator: func(ctx *console.Context, format string) error {
				if format != "" && format != "markdown" && format != "json" && format != "yaml" && format != "ansi" && format != "junit" {
					return errors.Errorf(`format "%s" does not exist (supported formats: markdown, ansi, json, junit, and yaml)`, format)
				}

				return nil
			},
		},
		&console.StringFlag{Name: "archive", DefaultValue: security.AdvisoryArchiveURL, Usage: "Advisory archive URL"},
		&console.BoolFlag{Name: "local", Usage: "Do not make HTTP calls (needs a valid cache file)"},
		&console.BoolFlag{Name: "no-dev", Usage: "Do not check packages listed under require-dev"},
		&console.BoolFlag{Name: "update-cache", Usage: "Update the cache (other flags are ignored)"},
		&console.BoolFlag{Name: "disable-exit-code", Usage: "Whether to fail when issues are detected"},
		&console.StringFlag{Name: "cache-dir", DefaultValue: os.TempDir(), Usage: "Cache directory"},
	},
	Action: func(c *console.Context) error {
		format := c.String("format")
		path := c.String("dir")
		advisoryArchiveURL := c.String("archive")

		db, err := security.NewDB(c.Bool("local"), advisoryArchiveURL, c.String("cache-dir"))
		if err != nil {
			return console.Exit(fmt.Sprintf("unable to load the advisory DB: %s", err), 127)
		}

		if c.Bool("update-cache") {
			return nil
		}

		lockReader, err := security.LocateLock(path)
		if err != nil {
			return console.Exit(err.Error(), 127)
		}

		lock, err := security.NewLock(lockReader)
		if err != nil {
			return console.Exit(fmt.Sprintf("unable to load the lock file: %s", err), 127)
		}

		vulns := security.Analyze(lock, db, c.Bool("no-dev"))

		output, err := security.Format(vulns, format)
		if err != nil {
			return console.Exit(fmt.Sprintf("unable to output the results: %s", err), 127)
		}
		terminal.Stdout.Write(output)

		if os.Getenv("GITHUB_WORKSPACE") != "" {
			gOutFile := os.Getenv("GITHUB_OUTPUT")

			f, err := os.OpenFile(gOutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return console.Exit(fmt.Sprintf("unable to open github output: %s", err), 127)
			}
			defer f.Close()

			// Ran inside a GitHub action, export vulns
			output, _ := security.Format(vulns, "raw_json")
			if _, err = f.WriteString("vulns=" + string(output) + "\n"); err != nil {
				return console.Exit(fmt.Sprintf("unable to write into github output: %s", err), 127)
			}
		}

		if vulns.Count() > 0 && !c.Bool(("disable-exit-code")) {
			return console.Exit("", 1)
		}
		return nil
	},
}
