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

package util

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

const confDir = "symfony5"

func GetHomeDir() string {
	return getUserHomeDir()
}

func getUserHomeDir() string {

	if InCloud() {
		u, err := user.Current()
		if err != nil {
			return filepath.Join(os.TempDir(), confDir)
		}
		return filepath.Join(os.TempDir(), u.Username, confDir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	// use the old path if it exists already
	legacy := filepath.Join(home, "."+confDir)
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		return legacy
	}

	// macos only: if $HOME/.config exist, prefer that over 'Library/Application Support'
	if runtime.GOOS == "darwin" {
		dotconf := filepath.Join(home, ".config")
		if _, err := os.Stat(dotconf); !os.IsNotExist(err) {
			return filepath.Join(dotconf, confDir)
		}
	}

	if userCfg, err := os.UserConfigDir(); err == nil {
		return filepath.Join(userCfg, confDir)
	}

	return legacy
}
