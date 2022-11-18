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
	"regexp"
	_ "unsafe"
)

// Temporary workaround allowing github.com/olekukonko/tablewriter to properly
// format tables with rows containing ANSI terminal links. A PR
// (https://github.com/olekukonko/tablewriter/pull/206) has been opened
// upstream, but we don't know if and when it will be merged. This Go
// compilation directive allows to access the unexported variable and update it
// with what we submitted upstream.
// To be removed once the PR is merged and released.
//
//go:linkname tableAnsiEscapingRegexp github.com/olekukonko/tablewriter.ansi
var tableAnsiEscapingRegexp = regexp.MustCompile("\033(?:\\[(?:[0-9]{1,3}(?:;[0-9]{1,3})*)?[m|K]|\\]8;;.*?\033\\\\)")
