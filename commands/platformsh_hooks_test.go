/*
 * Copyright (c) 2023-present Fabien Potencier <fabien@symfony.com>
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
	"flag"
	"strings"
	"testing"

	"github.com/symfony-cli/console"
)

func TestDeployHook(t *testing.T) {
	flags := flag.NewFlagSet("test", 0)
	flags.String("dir", "", "")
	c := console.NewContext(nil, flags, nil)

	for dir, expected := range map[string]string{
		"testdata/platformsh/version-mismatch-env/":    `The ".platform/services.yaml" file defines a "postgresql" version 14 database service but the ".env" file requires version 13.`,
		"testdata/platformsh/version-mismatch-config/": `The ".platform/services.yaml" file defines a "postgresql" version 14 database service but the "config/packages/doctrine.yaml" file requires version 13.`,
		"testdata/platformsh/ok/":                      ``,
		"testdata/platformsh/mariadb-version/":         ``,
		"testdata/platformsh/missing-version/":         `Set the "server_version" parameter to "14" in "config/packages/doctrine.yaml".`,
	} {
		flags.Set("dir", dir)
		err := upsunBeforeHooks["environment:push"](c)
		if err == nil {
			if expected != "" {
				t.Errorf("TestDeployHook(%q): got %v, expected %v", dir, err, expected)
			}
			continue
		}
		errString := strings.ReplaceAll(err.Error(), "\n", " ")
		if expected == "" {
			t.Errorf("TestDeployHook(%q): got %s, expected no errors", dir, errString)
		}
		if !strings.Contains(errString, expected) {
			t.Errorf("TestDeployHook(%q): got %s, expected %s", dir, errString, expected)
		}
	}
}
