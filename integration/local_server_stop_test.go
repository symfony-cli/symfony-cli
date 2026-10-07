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

func TestServerStop_WithDir(t *testing.T) {
	cmd := exec.Command("../symfony-cli", "server:start", "-d", "--dir=phpinfo/")
	err := cmd.Run()
	if err != nil {
		t.Errorf("Error starting server: %s", err)
	}

	// explicitly stop the server "contained" in the "phpinfo" directory
	cmd = exec.Command("../symfony-cli", "server:stop", "--dir=phpinfo/")
	err = cmd.Run()
	if err != nil {
		t.Errorf("Error stopping server: %s", err)
	}

	cmd = exec.Command("../symfony-cli", "server:list", "--no-ansi")
	output, _ := cmd.Output()

	if strings.Contains(string(output), "integration/phpinfo") {
		t.Errorf("Expected no servers to be running, got %s", output)

		Cleanup(t)
	}
}

func TestServerStop_WithAll(t *testing.T) {
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

	// only give the --all flag, which should stop all servers
	cmd := exec.Command("../symfony-cli", "server:stop", "--all")
	err := cmd.Run()
	if err != nil {
		t.Errorf("Error stopping server: %s", err)
	}

	cmd = exec.Command("../symfony-cli", "server:list", "--no-ansi")
	output, _ := cmd.Output()

	expectedToNotMatch := []string{
		"(.+)symfony-cli/integration/phpinfo/",
		"(.+)symfony-cli/integration/hello_world/",
	}

	for _, match := range expectedToNotMatch {
		matched, _ := regexp.Match(match, output)
		if matched {
			t.Errorf("Expected server NOT to be running while matching: %s", match)
		}
	}
}

func TestServerStop_CurrentDir(t *testing.T) {
	cmd := exec.Command("../symfony-cli", "server:start", "-d", "--dir=phpinfo/")

	err := cmd.Run()
	if err != nil {
		t.Errorf("Error starting server: %s", err)
	}

	cmd = exec.Command("../../symfony-cli", "server:stop")
	// change the working directory to the "phpinfo" directory so the server can be stopped
	// without needing to specify the --dir flag
	cmd.Dir = "phpinfo/"

	err = cmd.Run()
	if err != nil {
		t.Errorf("Error stopping server: %s", err)
	}

	cmd = exec.Command("../symfony-cli", "server:list", "--no-ansi")
	output, _ := cmd.Output()

	if strings.Contains(string(output), "symfony-cli/integration/phpinfo") {
		t.Errorf("Expected no servers to be running, got %s", output)
	}
}
