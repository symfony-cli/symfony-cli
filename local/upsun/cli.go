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

package upsun

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/util"
)

var (
	psh     *CLI
	pshOnce sync.Once
)

type CLI struct {
	Commands []*console.Command
	Hooks    map[string]console.BeforeFunc

	path string
}

func Get() (*CLI, error) {
	var err error
	pshOnce.Do(func() {
		psh, err = newCLI()
		if err != nil {
			err = errors.Wrap(err, "Unable to setup Platform.sh/Upsun CLI")
		}
	})
	return psh, err
}

func newCLI() (*CLI, error) {
	p := &CLI{
		Hooks: map[string]console.BeforeFunc{},
	}
	for _, command := range Commands {
		command.Action = p.proxyPSHCmd(strings.TrimPrefix(command.Category+":"+command.Name, "cloud:"))
		command.Args = []*console.Arg{
			{Name: "anything", Slice: true, Optional: true},
		}
		command.FlagParsing = console.FlagParsingSkipped
		command.Flags = append(command.Flags,
			&console.BoolFlag{Name: "no", Aliases: []string{"n"}},
			&console.BoolFlag{Name: "yes", Aliases: []string{"y"}},
		)
		p.Commands = append(p.Commands, command)
	}
	return p, nil
}

func (p *CLI) AddBeforeHook(name string, f console.BeforeFunc) {
	p.Hooks[name] = f
	for _, command := range p.Commands {
		if command.FullName() == name {
			command.FlagParsing = console.FlagParsingNormal
			break
		}
	}
}

func (p *CLI) getPath(brand CloudBrand) string {
	if p.path != "" {
		return p.path
	}

	home, err := homedir.Dir()
	if err != nil {
		panic("unable to get home directory")
	}

	// the Platform.sh CLI is always available on the containers thanks to the configurator
	p.path = filepath.Join(home, brand.BinaryPath())
	if !util.InCloud() {
		if cloudPath, err := Install(home, brand); err == nil {
			p.path = cloudPath
		}
	}
	return p.path
}

func (p *CLI) PSHMainCommands() []*console.Command {
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

func (p *CLI) proxyPSHCmd(commandName string) console.ActionFunc {
	return func(commandName string) console.ActionFunc {
		return func(ctx *console.Context) error {
			if hook, ok := p.Hooks[commandName]; ok && !console.IsHelp(ctx) {
				if err := hook(ctx); err != nil {
					return err
				}
			}
			brand := GuessCloudFromCommandName(ctx.Command.UserName)
			args := os.Args[1:]
			for i := range args {
				if args[i] == ctx.Command.UserName {
					args[i] = commandName
					break
				}
			}
			return p.executor(brand, args).Run()
		}
	}(commandName)
}

func (p *CLI) executor(brand CloudBrand, args []string) *exec.Cmd {
	prefix := brand.CLIPrefix

	env := []string{
		fmt.Sprintf("%sAPPLICATION_NAME=%s CLI for Symfony", prefix, brand),
		fmt.Sprintf("%sAPPLICATION_EXECUTABLE=symfony", prefix),
		"XDEBUG_MODE=off",
		fmt.Sprintf("%sWRAPPED=1", prefix),
	}
	if util.InCloud() {
		env = append(env, fmt.Sprintf("%sUPDATES_CHECK=0", prefix))
	}
	args[0] = strings.TrimPrefix(strings.TrimPrefix(args[0], "upsun:"), "cloud:")
	cmd := exec.Command(p.getPath(brand), args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func (p *CLI) RunInteractive(logger zerolog.Logger, projectDir string, args []string, debug bool, stdin io.Reader) (bytes.Buffer, bool) {
	var buf bytes.Buffer
	brand := GuessCloudFromCommandName(args[0])
	cmd := p.executor(brand, args)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	if debug {
		cmd.Stdout = io.MultiWriter(&buf, os.Stdout)
		cmd.Stderr = io.MultiWriter(&buf, os.Stderr)
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	if stdin != nil {
		cmd.Stdin = stdin
	}
	logger.Debug().Str("cmd", strings.Join(cmd.Args, " ")).Msgf("Executing %s CLI command interactively", GuessCloudFromCommandName(args[0]))
	if err := cmd.Run(); err != nil {
		return buf, false
	}
	return buf, true
}

func (p *CLI) WrapHelpPrinter() func(w io.Writer, templ string, data interface{}) {
	currentHelpPrinter := console.HelpPrinter
	return func(w io.Writer, templ string, data interface{}) {
		switch cmd := data.(type) {
		case *console.Command:
			if strings.HasPrefix(cmd.Category, "cloud") && cmd.FullName() != "cloud:deploy" {
				brand := GuessCloudFromCommandName(cmd.UserName)
				cmd := p.executor(brand, []string{cmd.FullName(), "--help"})
				cmd.Run()
			} else {
				currentHelpPrinter(w, templ, data)
			}
		default:
			currentHelpPrinter(w, templ, data)
		}
	}
}
