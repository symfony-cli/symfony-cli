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

package symfony

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/local/php"
)

type CliApp struct {
	Commands   []CliCommand
	Namespaces []CliNamespace
}

type CliNamespace struct {
	ID       string
	Commands []string
}

type CliCommand struct {
	Name        string
	Usage       []string
	Description string
	Help        string
	Definition  CliDefinition
	Hidden      bool
	Aliases     []string
}

type CliDefinition struct {
	Arguments map[string]CliArgument
	Options   map[string]CliOption
}

type CliArgument struct {
	Required    bool        `json:"is_required"`
	IsArray     bool        `json:"is_array"`
	Description string      `json:"description"`
	Default     interface{} `json:"default"`
}

type CliOption struct {
	Shortcut        string      `json:"shortcut"`
	Description     string      `json:"description"`
	AcceptValue     bool        `json:"accept_value"`
	IsValueRequired bool        `json:"is_value_required"`
	IsMultiple      bool        `json:"is_multiple"`
	Default         interface{} `json:"default"`
}

func NewCliApp(projectDir string, args []string) (*CliApp, error) {
	args = append(args, "list", "--format=json")
	var buf bytes.Buffer
	e := &php.Executor{
		BinName: "php",
		Dir:     projectDir,
		Args:    args,
		Stdout:  &buf,
		Stderr:  &buf,
	}
	if ret := e.Execute(false); ret != 0 {
		return nil, errors.Errorf("unable to list commands (%s):\n%s", strings.Join(args, " "), buf.String())
	}
	return parseCommands(buf.Bytes())
}

func NewGoCliApp(projectDir string, binPath string, args []string) (*CliApp, error) {
	var buf bytes.Buffer
	cmd := exec.Command(binPath, "list", "--format=json")
	cmd.Args = append(cmd.Args, args...)
	fmt.Println(cmd.Args)
	cmd.Dir = projectDir
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return nil, errors.Errorf("unable to list commands (%s):\n%s\n%s", strings.Join(args, " "), err, buf.String())
	}
	return parseCommands(buf.Bytes())
}

func parseCommands(output []byte) (*CliApp, error) {
	// Fix PHP types
	cleanOutput := bytes.ReplaceAll(output, []byte(`"arguments":[]`), []byte(`"arguments":{}`))
	var app *CliApp
	if err := json.Unmarshal(cleanOutput, &app); err != nil {
		return nil, err
	}
	return app, nil
}
