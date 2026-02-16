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
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/terminal"
	"mvdan.cc/sh/v3/syntax"
)

var variableExportCmd = &console.Command{
	Category: "var",
	Name:     "export",
	Usage:    "Export environment variables depending on the current context",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "multiline", Usage: "Display each export on its own line"},
		&console.BoolFlag{Name: "debug", Usage: "Debug Docker support"},
		&console.BoolFlag{Name: "quote", Usage: "Quote values as Bash strings"},
	},
	Args: []*console.Arg{
		{Name: "name", Optional: true, Description: "Print the value of this environment variable"},
	},
	Action: func(c *console.Context) error {
		dir := c.String("dir")
		if dir == "" {
			var err error
			if dir, err = os.Getwd(); err != nil {
				return err
			}
		}
		env, err := envs.GetEnv(dir, c.Bool("debug"))
		if err != nil {
			return err
		}

		var quote func(string) (string, error)

		if c.Bool("quote") {
			quote = func(s string) (string, error) { return syntax.Quote(s, syntax.LangBash) }
		} else {
			quote = func(s string) (string, error) { return s, nil }
		}

		if name := c.Args().Get("name"); name != "" {
			v, ok := envs.AsMap(env)[name]
			if !ok {
				return errors.Errorf("no environment variable with name %s", name)
			}
			q, err := quote(v)
			if err != nil {
				return errors.Errorf("can not quote value %q", v)
			}
			terminal.Print(q)
			return nil
		}

		if c.Bool("multiline") {
			m := envs.AsMap(env)
			keys := make([]string, 0)
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				q, err := quote(m[k])
				if err != nil {
					return errors.Errorf("can not quote value %q for environment variable %s", m[k], k)
				}
				terminal.Printfln("export %s=%s", k, q)
			}
		} else {
			var vars []string
			for k, v := range envs.AsMap(env) {
				q, err := quote(v)
				if err != nil {
					return errors.Errorf("can not quote value %q for environment variable %s", v, k)
				}
				vars = append(vars, fmt.Sprintf("%s=%s", k, q))
			}
			// output the string (useful when doing export $(envs))
			terminal.Print(strings.Join(vars, " "))
		}

		return nil
	},
}
