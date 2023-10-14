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
	Name          string
	Slug          string
	CommandPrefix string
	CLIPrefix     string
	GitRemoteName string
}

var UpsunBrand = CloudBrand{
	Name:          "Upsun",
	Slug:          "upsun",
	CommandPrefix: "upsun:",
	CLIPrefix:     "UPSUN_CLI_",
	GitRemoteName: "upsun",
}
var PlatformshBrand = CloudBrand{
	Name:          "Platform.sh",
	Slug:          "platformsh",
	CommandPrefix: "cloud:",
	CLIPrefix:     "PLATFORMSH_CLI_",
	GitRemoteName: "platform",
}

func (b CloudBrand) String() string {
	return b.Name
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
		if _, err := os.Stat(filepath.Join(dir, "."+brand.Slug)); err != nil {
			return brand
		}
	}
	return PlatformshBrand
}
