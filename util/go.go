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

package util

import (
	"os"
	"path/filepath"
	"strings"
)

func IsGoRun() bool {
	// Unfortunately, Golang does not expose that we are currently using go run
	// So we detect the main binary is (or used to be ;)) "go" and then the
	// current binary is within a temp "go-build" directory.
	_, exe := filepath.Split(os.Getenv("_"))
	argv0, _ := os.Executable()

	return exe == "go" && strings.Contains(argv0, "go-build")
}
