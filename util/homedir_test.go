/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/symfony-cli/terminal"
)

func TestLegacyConfigurationWarningDoesNotUseStandardOutput(t *testing.T) {
	if os.Getenv("SYMFONY_CLI_HOME_WARNING_HELPER") == "1" {
		GetHomeDir()
		terminal.Println("stdout remains available")
		os.Exit(0)
	}
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, LegacyConfigurationDirectory), 0755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=TestLegacyConfigurationWarningDoesNotUseStandardOutput")
	command.Env = append(os.Environ(),
		"SYMFONY_CLI_HOME_WARNING_HELPER=1",
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, "config"),
		"APPDATA="+filepath.Join(home, "AppData"),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "stdout remains available\n" {
		t.Fatalf("legacy warning changed standard output: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "The configuration location for the Symfony CLI has changed") {
		t.Fatalf("legacy warning was not written to standard error: %q", stderr.String())
	}
}
