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

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
	"golang.org/x/sync/errgroup"
)

var localServerStopCmd = &console.Command{
	Category: "local",
	Name:     "server:stop",
	Aliases:  []*console.Alias{{Name: "server:stop"}},
	Usage:    "Stop the local web server",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "all", Usage: "Stop all local web servers"},
	},
	Action: func(c *console.Context) error {
		if c.Bool("all") && c.IsSet("dir") {
			return fmt.Errorf("you cannot use the --all option with a specific directory")
		}

		var dirs []string

		if c.Bool("all") {
			configuredAndRunning, err := pid.ToConfiguredProjects(false)
			if err != nil {
				return err
			}

			for dir := range configuredAndRunning {
				dirs = append(dirs, dir)
			}
		} else {
			projectDir, err := getProjectDir(c.String("dir"))
			if err != nil {
				return err
			}

			dirs = append(dirs, projectDir)
		}

		return stopProjects(dirs, c.Bool("all"))
	},
}

func stopProjects(dirs []string, allFlag bool) error {
	ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
	running := 0

	if len(dirs) == 0 {
		ui.Success("No local web servers to stop")

		return nil
	}

	for _, dir := range dirs {
		projectDir, err := getProjectDir(dir)
		runningProcessesForProject := 0
		if err != nil {
			return err
		}

		if allFlag {
			ui.Section(fmt.Sprintf("Stopping project %s", projectDir))
		}

		webserver := pid.New(projectDir, nil)
		pids := append(pid.AllWorkers(projectDir), webserver)
		var g errgroup.Group
		for _, p := range pids {
			if !p.IsRunning() {
				continue
			}

			running++
			runningProcessesForProject++
			g.Go(p.WaitForExit)

			// we first notify the webserver in order to let it know it should
			// not restart any workers anymore
			if p.CustomName == pid.WebServerName {
				p.Signal(os.Interrupt)
				continue
			}
		}

		if runningProcessesForProject == 0 {
			ui.Success("The web server is not running")
			continue
		}

		for _, p := range pids {
			terminal.Printf("Stopping <comment>%s</>", p.ShortName())
			if !p.IsRunning() {
				terminal.Println(": <comment>not running</>")
				continue
			}

			// we don't "stop" the webserver because it acts as a monitoring
			// process and as such we already signaled it earlier (see previous
			// loop). If we do, the signal would be broadcast to the full
			// process group, breaking some workers (as docker compose for
			// example) because they would receive too many signals for a single
			// stop request.
			if p.CustomName == pid.WebServerName {
			} else if err := p.Stop(); err != nil {
				terminal.Printf(": <error>%s</>", err)
			}
			terminal.Println("")
		}

		terminal.Println("")
		if err := g.Wait(); err != nil {
			return err
		}
	}

	if running > 0 {
		ui.Success(fmt.Sprintf("Stopped %d process(es) successfully", running))
	}

	return nil
}
