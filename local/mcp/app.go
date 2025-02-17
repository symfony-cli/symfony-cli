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
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/local/php"
)

type Application struct {
	Commands []command
}

type command struct {
	Name        string
	Description string
	Help        string
	Definition  definition
	Hidden      bool
}

type definition struct {
	Arguments map[string]argument
	Options   map[string]option
}

type argument struct {
	Required    bool        `json:"is_required"`
	IsArray     bool        `json:"is_array"`
	Description string      `json:"description"`
	Default     interface{} `json:"default"`
}

type option struct {
	Description     string      `json:"description"`
	AcceptValue     bool        `json:"accept_value"`
	IsValueRequired bool        `json:"is_value_required"`
	IsMultiple      bool        `json:"is_multiple"`
	Default         interface{} `json:"default"`
}

func NewApp(projectDir string, args []string) (*Application, error) {
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

	// Fix PHP types
	cleanOutput := bytes.ReplaceAll(buf.Bytes(), []byte(`"arguments":[]`), []byte(`"arguments":{}`))

	var app *Application
	if err := json.Unmarshal(cleanOutput, &app); err != nil {
		return nil, err
	}

	return app, nil
}
