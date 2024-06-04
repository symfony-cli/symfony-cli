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

package php

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

func findComposerSystemSpecific(extraBin string) string {
	// Special Support for Scoop
	scoopPaths := []string{}

	if scoopPath, err := exec.LookPath("scoop"); err == nil {
		scoopPaths = append(scoopPaths, filepath.Dir(filepath.Dir(scoopPath)))
	}

	if scoopPath := os.Getenv("SCOOP"); scoopPath == "" {
		if homedir, err := homedir.Dir(); err != nil {
			scoopPaths = append(scoopPaths, filepath.Join(homedir, "scoop"))
		}
	}

	if scoopGlobalPath := os.Getenv("SCOOP_GLOBAL"); scoopGlobalPath == "" {
		if programData := os.Getenv("PROGRAMDATA"); programData != "" {
			scoopPaths = append(scoopPaths, filepath.Join(programData, "scoop"))
		}
	}

	for _, path := range scoopPaths {
		pharPath := filepath.Join(path, "apps", "composer", "current", "composer.phar")
		d, err := os.Stat(pharPath)
		if err != nil {
			continue
		}
		if m := d.Mode(); !m.IsDir() {
			// Yep!
			return pharPath
		}
	}

	return ""
}
