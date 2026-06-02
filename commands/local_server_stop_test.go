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
	"path/filepath"
	"testing"
)

func TestGetProjectDirAllowingMissing_DeletedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}

	projectDir, err := getProjectDirAllowingMissing(dir)
	if err != nil {
		t.Fatalf("getProjectDirAllowingMissing() error = %v", err)
	}

	expected, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	if projectDir != expected {
		t.Fatalf("getProjectDirAllowingMissing() = %q, want %q", projectDir, expected)
	}
}

func TestStopProjects_DeletedDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}

	if err := stopProjects([]string{dir}, false); err != nil {
		t.Fatalf("stopProjects() error = %v", err)
	}
}
