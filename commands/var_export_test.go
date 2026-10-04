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

import "testing"

func TestShellQuote(t *testing.T) {
	for _, tc := range []struct {
		value    string
		expected string
	}{
		{"", ""},
		{"dev", "dev"},
		{"postgres://app:secret@127.0.0.1:5432/app?serverVersion=16", "'postgres://app:secret@127.0.0.1:5432/app?serverVersion=16'"},
		{"mysql://127.0.0.1/app?a=1&b=2", "'mysql://127.0.0.1/app?a=1&b=2'"},
		{"two words", "'two words'"},
		{"$HOME", "'$HOME'"},
		{"it's", `'it'\''s'`},
		{"line1\nline2", "'line1\nline2'"},
	} {
		if actual := shellQuote(tc.value); actual != tc.expected {
			t.Errorf("shellQuote(%q) = %q, expected %q", tc.value, actual, tc.expected)
		}
	}
}
