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
	"slices"
	"testing"
)

func TestWatchedPaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "php.ini")
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}

	// relative paths are relative to the project directory, not the current one
	got := watchedPaths(dir, []string{dir, file, missing, "public", "php.ini"})
	want := []string{dir + string(filepath.Separator), file, missing, "public" + string(filepath.Separator), "php.ini"}
	if !slices.Equal(got, want) {
		t.Errorf("watchedPaths() = %q, want %q", got, want)
	}
}
