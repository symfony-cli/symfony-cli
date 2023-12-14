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

package platformsh

import (
	"os"
	"path/filepath"
	"strings"
)

type CloudBrand struct {
	Name              string
	ProjectConfigPath string
	CommandPrefix     string
	CLIConfigPath     string
	CLIPrefix         string
	GitRemoteName     string
	BinName           string
}

var UpsunBrand = CloudBrand{
	Name:              "Upsun",
	ProjectConfigPath: ".upsun",
	CLIConfigPath:     ".upsun-cli",
	CLIPrefix:         "UPSUN_CLI_",
	CommandPrefix:     "upsun:",
	GitRemoteName:     "upsun",
	BinName:           "upsun",
}
var PlatformshBrand = CloudBrand{
	Name:              "Platform.sh",
	ProjectConfigPath: ".platform",
	CLIConfigPath:     ".platformsh",
	CLIPrefix:         "PLATFORMSH_CLI_",
	CommandPrefix:     "cloud:",
	GitRemoteName:     "platform",
	BinName:           "platform",
}
var NoBrand = CloudBrand{}

func (b CloudBrand) String() string {
	return b.Name
}

// BinaryPath returns the cloud binary path.
func (b CloudBrand) BinaryPath() string {
	return filepath.Join(b.CLIConfigPath, "bin", b.BinName)
}

func GuessCloudFromCommandName(name string) CloudBrand {
	// if the namespace is upsun, then that's the brand we want to use
	if strings.HasPrefix(name, "upsun:") {
		return UpsunBrand
	}

	if dir, err := os.Getwd(); err == nil {
		return GuessCloudFromDirectory(dir)
	}

	return PlatformshBrand
}

func GuessCloudFromDirectory(dir string) CloudBrand {
	for _, brand := range []CloudBrand{UpsunBrand, PlatformshBrand} {
		if _, err := os.Stat(filepath.Join(dir, brand.ProjectConfigPath)); err == nil {
			return brand
		}
	}
	return NoBrand
}
