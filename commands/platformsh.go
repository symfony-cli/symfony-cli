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
	"bytes"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/symfony-cli/util"
)

type platformshCLI struct {
	Commands []*console.Command

	path string
}

func NewPlatformShCLI() (*platformshCLI, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	p := &platformshCLI{
		path: filepath.Join(home, ".platformsh", "bin", "platform"),
	}
	for _, command := range platformsh.Commands {
		command.Action = p.proxyPSHCmd(strings.TrimPrefix(command.Category+":"+command.Name, "cloud:"))
		command.Args = []*console.Arg{
			{Name: "anything", Slice: true, Optional: true},
		}
		command.Flags = append(command.Flags,
			&console.BoolFlag{Name: "no", Aliases: []string{"n"}},
			&console.BoolFlag{Name: "yes", Aliases: []string{"y"}},
		)
		if _, ok := platformshBeforeHooks[command.FullName()]; !ok {
			// do not parse flags if we don't have hooks
			command.FlagParsing = console.FlagParsingSkipped
		}
		p.Commands = append(p.Commands, command)
	}
	return p, nil
}

func (p *platformshCLI) PSHMainCommands() []*console.Command {
	names := map[string]bool{
		"cloud:project:list":       true,
		"cloud:environment:list":   true,
		"cloud:environment:branch": true,
		"cloud:tunnel:open":        true,
		"cloud:environment:ssh":    true,
		"cloud:environment:push":   true,
		"cloud:domain:list":        true,
		"cloud:variable:list":      true,
		"cloud:user:add":           true,
	}
	mainCmds := []*console.Command{}
	for _, command := range p.Commands {
		if names[command.FullName()] {
			mainCmds = append(mainCmds, command)
		}
	}
	return mainCmds
}

func (p *platformshCLI) proxyPSHCmd(commandName string) console.ActionFunc {
	return func(commandName string) console.ActionFunc {
		return func(c *console.Context) error {
			// the Platform.sh CLI is always available on the containers thanks to the configurator
			if !util.InCloud() {
				home, err := homedir.Dir()
				if err != nil {
					return errors.WithStack(err)
				}
				if err := php.InstallPlatformPhar(home); err != nil {
					return console.Exit(err.Error(), 1)
				}
			}

			if hook, ok := platformshBeforeHooks["cloud:"+commandName]; ok && !console.IsHelp(c) {
				if err := hook(c); err != nil {
					return errors.WithStack(err)
				}
			}

			args := os.Args[1:]
			for i := range args {
				if args[i] == c.Command.UserName {
					args[i] = commandName
					break
				}
			}
			e := p.executor(args)
			return console.Exit("", e.Execute(false))
		}
	}(commandName)
}

func (p *platformshCLI) executor(args []string) *php.Executor {
	env := []string{
		"PLATFORMSH_CLI_APPLICATION_NAME=Platform.sh CLI for Symfony",
		"PLATFORMSH_CLI_APPLICATION_EXECUTABLE=symfony",
		"XDEBUG_MODE=off",
		"PLATFORMSH_CLI_WRAPPED=1",
	}
	if util.InCloud() {
		env = append(env, "PLATFORMSH_CLI_UPDATES_CHECK=0")
	}
	e := &php.Executor{
		BinName:  "php",
		Args:     append([]string{"php", p.path}, args...),
		ExtraEnv: env,
	}
	e.Paths = append([]string{filepath.Dir(p.path)}, e.Paths...)
	return e
}

func (p *platformshCLI) RunInteractive(logger zerolog.Logger, projectDir string, args []string, debug bool, stdin io.Reader) (bytes.Buffer, bool) {
	var buf bytes.Buffer

	e := p.executor(args)
	if projectDir != "" {
		e.Dir = projectDir
	}
	if debug {
		e.Stdout = io.MultiWriter(&buf, os.Stdout)
		e.Stderr = io.MultiWriter(&buf, os.Stderr)
	} else {
		e.Stdout = &buf
		e.Stderr = &buf
	}
	if stdin != nil {
		e.Stdin = stdin
	}
	logger.Debug().Str("cmd", strings.Join(e.Args, " ")).Msg("Executing Platform.sh CLI command interactively")
	if ret := e.Execute(false); ret != 0 {
		return buf, false
	}
	return buf, true
}

func (p *platformshCLI) WrapHelpPrinter() func(w io.Writer, templ string, data interface{}) {
	currentHelpPrinter := console.HelpPrinter
	return func(w io.Writer, templ string, data interface{}) {
		switch cmd := data.(type) {
		case *console.Command:
			if strings.HasPrefix(cmd.Category, "cloud") {
				e := p.executor([]string{strings.TrimPrefix(cmd.FullName(), "cloud:"), "--help", "--ansi"})
				e.Execute(false)
			} else {
				currentHelpPrinter(w, templ, data)
			}
		default:
			currentHelpPrinter(w, templ, data)
		}
	}
}
