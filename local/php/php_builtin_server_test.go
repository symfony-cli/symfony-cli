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
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPhpRouterExposesCliBinaryName(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP is not installed")
	}
	dir := t.TempDir()
	files := map[string][]byte{
		"router.php":          phprouter,
		"router.php-test-env": []byte(`<?php $_ENV['SYMFONY_CLI_BINARY_NAME'] = 'symfony'; $_ENV['APP_FRONT_CONTROLLER'] = 'index.php';`),
		"index.php":           []byte("<?php"),
		"direct.php":          []byte("<?php"),
		"probe.php": []byte(`<?php
$_SERVER['DOCUMENT_ROOT'] = __DIR__;
$_SERVER['SCRIPT_NAME'] = $argv[1];
$_SERVER['HTTP___SYMFONY_LOCAL_REQUEST_ID__'] = 'test';
$_SERVER['SYMFONY_CLI_BINARY_NAME'] = 'stale-cli';
$handled = require __DIR__.'/router.php';
echo json_encode(['binary' => $_SERVER['SYMFONY_CLI_BINARY_NAME'] ?? null, 'handled' => false !== $handled]);
`),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, name), contents, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"/route", "/direct.php"} {
		t.Run(path, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, php, "-n", "-d", "memory_limit=64M", filepath.Join(dir, "probe.php"), path)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("PHP router: %v\n%s", err, output)
			}
			var result struct {
				Binary  string `json:"binary"`
				Handled bool   `json:"handled"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("invalid router response: %v\n%s", err, output)
			}
			if result.Binary != "symfony" {
				t.Errorf("CLI binary name = %q, want symfony", result.Binary)
			}
			if expected := path == "/route"; result.Handled != expected {
				t.Errorf("router handled request = %t, want %t", result.Handled, expected)
			}
		})
	}
}
