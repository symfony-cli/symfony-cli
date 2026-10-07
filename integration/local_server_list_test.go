//go:build integration
// +build integration

/*
 * Copyright (c) 2024-present Fabien Potencier <fabien@symfony.com>
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

package integration

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestServerList(t *testing.T) {
	startCommands := []string{
		"../symfony-cli server:start -d --dir=phpinfo/",
		"../symfony-cli server:start -d --dir=hello_world/",
	}

	for _, command := range startCommands {
		cmd := exec.Command(strings.Split(command, " ")[0], strings.Split(command, " ")[1:]...)
		err := cmd.Run()
		if err != nil {
			t.Errorf("Error running command: %s", err)
		}
	}

	cmd := exec.Command("../symfony-cli", "server:list", "--no-ansi")
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("Error listing servers: %s", err)
	}

	expectedMatches := []string{
		"(.+)symfony-cli/integration/phpinfo/ | 8000",
		"(.+)symfony-cli/integration/hello_world/ | 8001",
	}

	for _, match := range expectedMatches {
		matched, _ := regexp.Match(match, output)
		if !matched {
			t.Errorf("Expected server to be running while matching: %s", match)
		}
	}

	Cleanup(t)
}
