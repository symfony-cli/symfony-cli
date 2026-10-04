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

func TestLocalServerURL(t *testing.T) {
	for _, tc := range []struct {
		listenIp string
		want     string
	}{
		{listenIp: "127.0.0.1", want: "https://127.0.0.1:8000"},
		{listenIp: "", want: "https://127.0.0.1:8000"},
		{listenIp: "0.0.0.0", want: "https://127.0.0.1:8000"},
		{listenIp: "::", want: "https://127.0.0.1:8000"},
		{listenIp: "localhost", want: "https://127.0.0.1:8000"},
		{listenIp: "192.168.1.10", want: "https://192.168.1.10:8000"},
		{listenIp: "::1", want: "https://[::1]:8000"},
	} {
		if got := localServerURL("https", tc.listenIp, 8000); got != tc.want {
			t.Errorf("localServerURL(%q) = %q, want %q", tc.listenIp, got, tc.want)
		}
	}
}
