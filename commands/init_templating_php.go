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

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/symfony-cli/terminal"
)

func hasComposerPackage(directory, pkg string) bool {
	lock, err := parseComposerLock(directory)
	if err == nil {
		for _, p := range lock.Packages {
			if p.Name == pkg {
				return true
			}
		}

		for _, p := range lock.PackagesDev {
			if p.Name == pkg {
				return true
			}
		}

		return false
	}

	json, err2 := parseComposerJSON(directory)
	if err2 == nil {
		if _, ok := json.Require[pkg]; ok {
			return true
		}

		if _, ok := json.RequireDev[pkg]; ok {
			return true
		}

		return false
	}

	terminal.Logger.Warn().Msg(err.Error())
	terminal.Logger.Warn().Msg(err2.Error())

	return false
}

func phpExtensions(directory string) []string {
	exts := []string{}
	seen := map[string]bool{}

	lock, err := parseComposerLock(directory)
	if err == nil {
		for ext := range lock.Platform {
			if strings.HasPrefix(ext, "ext-") {
				exts = append(exts, ext[4:])
				seen[ext[4:]] = true
			}
		}
	}

	json, err2 := parseComposerJSON(directory)
	if err2 == nil {
		for ext := range json.Require {
			if strings.HasPrefix(ext, "ext-") && !seen[ext[4:]] {
				exts = append(exts, ext[4:])
				seen[ext[4:]] = true
			}
		}

		for ext := range json.RequireDev {
			if strings.HasPrefix(ext, "ext-") && !seen[ext[4:]] {
				exts = append(exts, ext[4:])
				seen[ext[4:]] = true
			}
		}
	}

	if err != nil {
		terminal.Logger.Warn().Msg(err.Error())
	}
	if err2 != nil {
		terminal.Logger.Warn().Msg(err2.Error())
	}

	return exts
}

func hasPHPExtension(directory, ext string) bool {
	if !strings.HasPrefix(ext, "ext-") {
		ext = fmt.Sprintf("ext-%s", ext)
	}

	lock, err := parseComposerLock(directory)
	if err == nil {
		_, ok := lock.Platform[ext]

		return ok
	}

	json, err2 := parseComposerJSON(directory)
	if err2 == nil {
		if _, ok := json.Require[ext]; ok {
			return true
		}

		if _, ok := json.RequireDev[ext]; ok {
			return true
		}

		return false
	}

	terminal.Logger.Warn().Msg(err.Error())
	terminal.Logger.Warn().Msg(err2.Error())

	return false
}

type composerLock struct {
	Platform          map[string]string `json:"platform"`
	PlatformOverrides map[string]string `json:"platform-overrides"`
	Packages          []struct {
		Name, Version string
	} `json:"packages"`
	PackagesDev []struct {
		Name, Version string
	} `json:"packages-dev"`
}

func parseComposerLock(directory string) (*composerLock, error) {
	b, err := os.ReadFile(filepath.Join(directory, "composer.lock"))
	if err != nil {
		return nil, err
	}

	var lock composerLock

	if err := json.Unmarshal(b, &lock); err != nil {
		return nil, err
	}

	return &lock, err
}

type composerJSON struct {
	Config struct {
		Platform map[string]string `json:"platform"`
	} `json:"config"`
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

func parseComposerJSON(directory string) (*composerJSON, error) {
	b, err := os.ReadFile(filepath.Join(directory, "composer.json"))
	if err != nil {
		return nil, err
	}

	var composerJSON composerJSON
	if err := json.Unmarshal(b, &composerJSON); err != nil {
		return nil, err
	}

	return &composerJSON, nil
}
