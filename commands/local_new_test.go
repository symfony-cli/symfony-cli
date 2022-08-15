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
	"testing"
)

func TestParseDockerComposeServices(t *testing.T) {
	for dir, expected := range map[string]CloudService{
		"testdata/postgresql/noversion/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "13",
		},
		"testdata/postgresql/10/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "10",
		},
		"testdata/postgresql/14/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "13",
		},
	} {
		result := parseDockerComposeServices(dir)
		if result[0].Version != expected.Version {
			t.Errorf("parseDockerComposeServices(none/%q): got %v, expected %v", dir, result[0].Version, expected.Version)
		}
	}
}
