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

package main

//go:generate go run ./local/platformsh/generator/...

import (
	"fmt"
	"os"
	"time"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/commands"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/terminal"
)

var (
	// version is overridden at linking time
	version = "dev"
	// channel is overridden at linking time
	channel = "dev"
	// overridden at linking time
	buildDate string
)

func getCliExtraEnv() []string {
	return []string{
		"SYMFONY_CLI_VERSION=" + version,
		"SYMFONY_CLI_BINARY_NAME=" + console.CurrentBinaryName(),
	}
}

func main() {
	if os.Getenv("SC_DEBUG") == "1" {
		terminal.SetLogLevel(5)
	}

	args := os.Args
	name := console.CurrentBinaryName()
	// called via "php"?
	if php.IsBinaryName(name) {
		fmt.Printf(`Using the Symfony wrappers to call PHP is not possible anymore; remove the wrappers and use "symfony %s" instead.`, name)
		fmt.Println()
		os.Exit(1)
	}
	// called via "symfony php"?
	if len(args) >= 2 && php.IsBinaryName(args[1]) {
		e := &php.Executor{
			BinName:  args[1],
			Args:     args[1:],
			ExtraEnv: getCliExtraEnv(),
			Logger:   terminal.Logger,
		}
		os.Exit(e.Execute(true))
	}
	// called via "symfony console"?
	if len(args) >= 2 && args[1] == "console" {
		if executor, err := php.SymfonyConsoleExecutor(terminal.Logger, args[2:]); err == nil {
			executor.ExtraEnv = getCliExtraEnv()
			os.Exit(executor.Execute(false))
		}
	}
	// called via "symfony composer" or "symfony pie"?
	if len(args) >= 2 {
		if args[1] == "composer" {
			res := php.Composer("", args[2:], getCliExtraEnv(), os.Stdout, os.Stderr, os.Stderr, terminal.Logger)
			terminal.Eprintln(res.Error())
			os.Exit(res.ExitCode())
		}

		if args[1] == "pie" {
			res := php.Pie("", args[2:], getCliExtraEnv(), os.Stdout, os.Stderr, os.Stderr, terminal.Logger)
			terminal.Eprintln(res.Error())
			os.Exit(res.ExitCode())
		}
	}

	for _, env := range []string{"BRANCH", "ENV", "APPLICATION_NAME"} {
		if os.Getenv("SYMFONY_"+env) != "" {
			continue
		}

		if v := os.Getenv("PLATFORM_" + env); v != "" {
			os.Setenv("SYMFONY_"+env, v)
			continue
		}
	}

	cmds := commands.CommonCommands()
	psh, err := platformsh.Get()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	cmds = append(cmds, psh.Commands...)
	console.HelpPrinter = psh.WrapHelpPrinter()
	app := &console.Application{
		Name:          "Symfony CLI",
		Usage:         "Symfony CLI helps developers manage projects, from local code to remote infrastructure",
		Copyright:     fmt.Sprintf("(c) 2021-%d Fabien Potencier", time.Now().Year()),
		FlagEnvPrefix: []string{"SYMFONY", "PLATFORM"},
		Commands:      cmds,
		Action: func(ctx *console.Context) error {
			if ctx.Args().Len() == 0 {
				return commands.WelcomeAction(ctx)
			}
			return console.ShowAppHelpAction(ctx)
		},
		Before:    commands.InitAppFunc,
		Version:   version,
		Channel:   channel,
		BuildDate: buildDate,
	}
	app.Run(args)
}
