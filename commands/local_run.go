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
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

var localRunCmd = &console.Command{
	Category: "local",
	Name:     "run",
	Aliases:  []*console.Alias{{Name: "run"}},
	Usage:    "Run a program with environment variables set depending on the current context",
	Flags: []console.Flag{
		&console.BoolFlag{Name: "daemon", Aliases: []string{"d"}, Usage: "Run the command in the background"},
		&console.StringSliceFlag{
			Name:  "watch",
			Usage: "Restart command when some change happens on this file or in this directory (recursively)",
		},
	},
	FlagParsing: console.FlagParsingSkippedAfterFirstArg,
	Args: []*console.Arg{
		{Name: "bin"},
		{Name: "args", Optional: true, Slice: true},
	},
	Action: func(c *console.Context) error {
		directories := make([]string, 0, len(c.StringSlice("watch")))
		for _, directory := range c.StringSlice("watch") {
			directories = append(directories, strings.Split(directory, ",")...)
		}
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return errors.WithStack(err)
		}

		mode := local.RunnerModeOnce
		if c.Bool("daemon") {
			mode = local.RunnerModeLoopDetached
		}
		pidFile := pid.New(projectDir, append([]string{c.Args().Get("bin")}, c.Args().Tail()...))
		if pidFile.IsRunning() {
			return errors.Errorf("Unable to start the command: it is already running for this project as PID %d", pidFile.Pid)
		}

		pidFile.Watched = directories
		runner, err := local.NewRunner(pidFile, mode)
		if err != nil {
			return errors.WithStack(err)
		}

		runner.BuildCmdHook = func(cmd *exec.Cmd) error {
			env, err := envs.GetEnv(pidFile.Dir, terminal.IsDebug())
			if err != nil {
				return errors.WithStack(err)
			}

			cmd.Env = append(cmd.Env, envs.AsSlice(env)...)

			return nil
		}

		if err := runner.Run(); err != nil {
			if _, wentToBackground := err.(local.RunnerWentToBackground); wentToBackground {
				terminal.Printfln("Stream the logs via <info>%s server:log</>", c.App.HelpName)
				return nil
			}

			return errors.WithStack(err)
		}

		return nil
	},
}
