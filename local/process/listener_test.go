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

package process

import (
	"net"
	"testing"
)

func TestCreateListener(t *testing.T) {
	for _, tc := range []struct {
		name     string
		listenIp string
		wantIP   string
	}{
		{name: "IPv4 loopback", listenIp: "127.0.0.1", wantIP: "127.0.0.1"},
		{name: "IPv6 loopback", listenIp: "::1", wantIP: "::1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if probe, err := net.Listen("tcp", net.JoinHostPort(tc.listenIp, "0")); err != nil {
				t.Skipf("%s is not available: %v", tc.listenIp, err)
			} else {
				probe.Close()
			}

			ln, port, err := CreateListener(tc.listenIp, 0, 0)
			if err != nil {
				t.Fatalf("CreateListener(%q) returned an error: %v", tc.listenIp, err)
			}
			defer ln.Close()

			addr := ln.Addr().(*net.TCPAddr)
			if got := addr.IP.String(); got != tc.wantIP {
				t.Errorf("listening on %s, want %s", got, tc.wantIP)
			}
			if addr.Port != port {
				t.Errorf("returned port %d, want %d", port, addr.Port)
			}
		})
	}
}
