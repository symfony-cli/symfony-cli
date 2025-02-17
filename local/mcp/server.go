/*
 * Copyright (c) 2025-present Fabien Potencier <fabien@symfony.com>
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

package mcp

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/symfony-cli/symfony-cli/local/php"
)

type MCP struct {
	server     *server.MCPServer
	apps       map[string]*Application
	appArgs    map[string][]string
	projectDir string
}

var excludedCommands = map[string]bool{
	"list":       true,
	"_complete":  true,
	"completion": true,
}

var excludedOptions = map[string]bool{
	"help":           true,
	"silent":         true,
	"quiet":          true,
	"verbose":        true,
	"version":        true,
	"ansi":           true,
	"no-ansi":        true,
	"env":            true,
	"format":         true,
	"no-interaction": true,
	"no-debug":       true,
	"profile":        true,
}

func NewServer(projectDir string) (*MCP, error) {
	mcp := &MCP{
		projectDir: projectDir,
		apps:       map[string]*Application{},
	}

	mcp.server = server.NewMCPServer(
		"Symfony CLI Server",
		"1.0.0",
		server.WithLogging(),
		server.WithResourceCapabilities(true, true),
	)

	mcp.appArgs = map[string][]string{
		"symfony": {"php", "bin/console"},
		//		"cloud":    {"run", "upsun"},
	}

	e := &php.Executor{
		Dir:     projectDir,
		BinName: "php",
	}
	if composerPath, err := e.FindComposer(""); err == nil {
		mcp.appArgs["composer"] = []string{"php", composerPath}
	}

	for name, args := range mcp.appArgs {
		var err error
		mcp.apps[name], err = NewApp(projectDir, args)
		if err != nil {
			return nil, err
		}
		for _, command := range mcp.apps[name].Commands {
			if _, ok := excludedCommands[command.Name]; ok {
				continue
			}
			if command.Hidden {
				continue
			}
			if err := mcp.addTool(name, command); err != nil {
				return nil, err
			}
		}
	}

	return mcp, nil
}

func (p *MCP) Start() error {
	return server.ServeStdio(p.server)
}

func (p *MCP) addTool(appName string, cmd command) error {
	toolOptions := []mcp.ToolOption{}
	toolOptions = append(toolOptions, mcp.WithDescription(cmd.Description+"\n\n"+cmd.Help))
	for name, arg := range cmd.Definition.Arguments {
		argOptions := []mcp.PropertyOption{
			mcp.Description(arg.Description),
		}
		if arg.Required {
			argOptions = append(argOptions, mcp.Required())
		}
		toolOptions = append(toolOptions, mcp.WithString("arg_"+name, argOptions...))
	}
	for name, option := range cmd.Definition.Options {
		if _, ok := excludedOptions[name]; ok {
			continue
		}
		optOptions := []mcp.PropertyOption{
			mcp.Description(option.Description),
		}
		if option.AcceptValue {
			toolOptions = append(toolOptions, mcp.WithString("opt_"+name, optOptions...))
		} else {
			toolOptions = append(toolOptions, mcp.WithBoolean("opt_"+name, optOptions...))
		}
	}

	toolName := appName + "--" + strings.ReplaceAll(cmd.Name, ":", "-")
	regexp := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	if !regexp.MatchString(toolName) {
		return fmt.Errorf("invalid command name: %s", cmd.Name)
	}

	tool := mcp.NewTool(toolName, toolOptions...)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executorArgs := []string{cmd.Name}
		for name, value := range request.Params.Arguments {
			if strings.HasPrefix(name, "arg_") {
				arg, ok := value.(string)
				if !ok {
					return mcp.NewToolResultError(fmt.Sprintf("argument value for \"%s\" must be a string", name)), nil
				}
				executorArgs = append(executorArgs, arg)
			} else if strings.HasPrefix(name, "opt_") {
				if cmd.Definition.Options[strings.TrimPrefix(name, "opt_")].AcceptValue {
					arg, ok := value.(string)
					if !ok {
						return mcp.NewToolResultError(fmt.Sprintf("argument value for \"%s\" must be a string", name)), nil
					}
					executorArgs = append(executorArgs, fmt.Sprintf("--%s=%s", strings.TrimPrefix(name, "opt_"), arg))
				} else {
					arg, ok := value.(bool)
					if !ok {
						return mcp.NewToolResultError(fmt.Sprintf("argument value for \"%s\" must be a string", name)), nil
					}
					if arg {
						executorArgs = append(executorArgs, fmt.Sprintf("--%s", strings.TrimPrefix(name, "opt_")))
					}
				}
			} else {
				return mcp.NewToolResultText(fmt.Sprintf("Unknown argument: %s", name)), nil
			}
		}
		executorArgs = append(executorArgs, "--no-ansi")
		executorArgs = append(executorArgs, "--no-interaction")
		var buf bytes.Buffer
		e := &php.Executor{
			BinName: "php",
			Dir:     p.projectDir,
			Args:    append(p.appArgs[appName], executorArgs...),
			Stdout:  &buf,
			Stderr:  &buf,
		}
		if ret := e.Execute(false); ret != 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Error running %s (exit code: %d)\n%s", strings.Join(executorArgs, " "), ret, buf.String())), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	}

	p.server.AddTool(tool, handler)

	return nil
}
