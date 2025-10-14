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
	"os"
	"path/filepath"
	"strings"
)

type CloudProduct struct {
	Name              string
	ProjectConfigPath string
	CommandPrefix     string
	CLIConfigPath     string
	CLIPrefix         string
	GitRemoteName     string
	BinName           string
}

var Flex = CloudProduct{
	Name:              "Upsun Flex",
	ProjectConfigPath: ".upsun",
	CLIConfigPath:     ".upsun-cli",
	CLIPrefix:         "UPSUN_CLI_",
	CommandPrefix:     "upsun:",
	GitRemoteName:     "upsun",
	BinName:           "upsun",
}
var Fixed = CloudProduct{
	Name:              "Upsun Fixed",
	ProjectConfigPath: ".platform",
	CLIConfigPath:     ".platformsh",
	CLIPrefix:         "PLATFORMSH_CLI_",
	CommandPrefix:     "cloud:",
	GitRemoteName:     "platform",
	BinName:           "platform",
}

// NoProduct is used when there is no explicit setting for the product.
var NoProduct = CloudProduct{
	Name:              "",
	ProjectConfigPath: "",
	CLIConfigPath:     ".platformsh",
	CLIPrefix:         "PLATFORMSH_CLI_",
	CommandPrefix:     "cloud:",
	GitRemoteName:     "",
	BinName:           "platform",
}

func (b CloudProduct) String() string {
	return b.Name
}

// BinaryPath returns the cloud binary path.
func (b CloudProduct) BinaryPath() string {
	return filepath.Join(b.CLIConfigPath, "bin", b.BinName)
}

func GuessProductFromCommandName(name string) CloudProduct {
	// if the namespace is upsun, then that's the product we want to use
	if strings.HasPrefix(name, "upsun:") {
		return Flex
	}

	if dir, err := os.Getwd(); err == nil {
		return GuessProductFromDirectory(dir)
	}

	return Fixed
}

func GuessProductFromDirectory(dir string) CloudProduct {
	for _, product := range []CloudProduct{Flex, Fixed} {
		if _, err := os.Stat(filepath.Join(dir, product.ProjectConfigPath)); err == nil {
			return product
		}
	}
	return NoProduct
}
