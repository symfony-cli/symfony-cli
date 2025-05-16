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
	legacyPath := filepath.Join(home, "."+confDir)
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		return legacyPath
	}

	if userCfg, err := os.UserConfigDir(); err == nil {
		return filepath.Join(userCfg, confDir)
	}

	return filepath.Join(".", "."+confDir)
}
