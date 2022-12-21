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
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/commands"
	"github.com/symfony-cli/symfony-cli/local/php"
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
		}
		os.Exit(e.Execute(true))
	}
	// called via "symfony console"?
	if len(args) >= 2 && args[1] == "console" {
		args[1] = "bin/console"
		if _, err := os.Stat("app/console"); err == nil {
			args[1] = "app/console"
		}
		e := &php.Executor{
			BinName:  "php",
			Args:     args,
			ExtraEnv: getCliExtraEnv(),
		}
		os.Exit(e.Execute(false))
	}
	// called via "symfony composer"?
	if len(args) >= 2 && args[1] == "composer" {
		res := php.Composer("", args[2:], getCliExtraEnv(), os.Stdout, os.Stderr, io.Discard, zerolog.Nop())
		terminal.Eprintln(res.Error())
		os.Exit(res.ExitCode())
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
	psh, err := commands.GetPSH()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	cmds = append(cmds, psh.Commands...)
	console.HelpPrinter = psh.WrapHelpPrinter()
	app := &console.Application{
		Name:          "Symfony CLI",
		Usage:         "Symfony CLI helps developers manage projects, from local code to remote infrastructure",
		Copyright:     fmt.Sprintf("(c) 2021-%d Fabien Potencier <bg=#0057B7;fg=#FFDD00>#StandWith</><bg=#FFDD00;fg=#0057B7>Ukraine</> <href=https://sf.to/ukraine>Support Ukraine</>", time.Now().Year()),
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
	err = app.Run(args)
	if err != nil {
		panic(err)
	}
}
