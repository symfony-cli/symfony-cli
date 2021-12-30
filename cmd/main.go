package main

//go:generate go run ../local/platformsh/platformsh_config_generator/main.go

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/cmd/commands"
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
			BinName: args[1],
			Args:    args[1:],
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
			BinName: "php",
			Args:    args,
		}
		os.Exit(e.Execute(false))
	}
	// called via "symfony composer"?
	if len(args) >= 2 && args[1] == "composer" {
		res := php.Composer("", args[2:], os.Stdout, os.Stderr, ioutil.Discard)
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
	cmds = append(cmds, psh.PSHCommands()...)
	console.HelpPrinter = psh.WrapHelpPrinter()
	app := &console.Application{
		Name:          "Symfony CLI",
		Usage:         "Symfony CLI helps developers manage projects, from local code to remote infrastructure",
		Copyright:     fmt.Sprintf("(c) 2017-%d Symfony SAS", time.Now().Year()),
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
