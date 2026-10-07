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
	"net/http"
	"os/exec"
	"testing"
)

func TestServerStartDaemon(t *testing.T) {
	cmd := exec.Command("../symfony-cli", "server:start", "-d", "--dir=phpinfo/")
	err := cmd.Run()
	if err != nil {
		t.Errorf("Error running command: %s", err)
	}

	r, err := http.Head("http://localhost:8000")
	if err != nil {
		t.Errorf("Error sending request: %s", err)
	}

	if r.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", r.StatusCode)
	}

	Cleanup(t)
}
