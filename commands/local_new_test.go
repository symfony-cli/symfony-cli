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
	"os"
	"strconv"
	"testing"

	"github.com/symfony-cli/symfony-cli/local/platformsh"
)

func TestParseDockerComposeServices(t *testing.T) {
	lastVersion := platformsh.ServiceLastVersion("postgresql")

	if n, err := strconv.Atoi(lastVersion); err != nil {
		t.Error("Could not generate test cases:", err)
	} else {
		os.Setenv("POSTGRES_NEXT_VERSION", strconv.Itoa(n+1))
		defer os.Unsetenv("POSTGRES_NEXT_VERSION")
	}

	for dir, expected := range map[string]CloudService{
		"testdata/postgresql/noversion/": {
			Name:    "database",
			Type:    "postgresql",
			Version: lastVersion,
		},
		"testdata/postgresql/10/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "10",
		},
		"testdata/postgresql/next/": {
			Name:    "database",
			Type:    "postgresql",
			Version: lastVersion,
		},
	} {
		result := parseDockerComposeServices(dir)
		if result[0].Version != expected.Version {
			t.Errorf("parseDockerComposeServices(none/%q): got %v, expected %v", dir, result[0].Version, expected.Version)
		}
	}
}
