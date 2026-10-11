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

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestScheme(t *testing.T) {
	for _, tc := range []struct {
		name, remoteAddr, forwardedProto, want string
	}{
		{"plain HTTP", "127.0.0.1:1234", "", "http"},
		{"local TLS terminator", "127.0.0.1:1234", "https", "https"},
		{"spoofed by a remote client", "192.0.2.1:1234", "https", "http"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.forwardedProto != "" {
				r.Header.Set("X-Forwarded-Proto", tc.forwardedProto)
			}
			if got := RequestScheme(r); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
