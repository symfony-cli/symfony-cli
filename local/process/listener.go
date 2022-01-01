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
	"strconv"

	"github.com/pkg/errors"
)

// CreateListener creates a listener on a port
// Pass a prefered port (will increment by 1 if port is not available)
// or pass 0 to auto-find any available port
func CreateListener(preferedPort int) (net.Listener, int, error) {
	var ln net.Listener
	var err error
	port := preferedPort
	max := 50
	for {
		// we really want to test availability on 127.0.0.1
		ln, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			ln.Close()
			// but then, we want to listen to as many local IP's as possible
			ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
			if err == nil {
				break
			}
		}
		if preferedPort == 0 {
			return nil, 0, errors.Wrap(err, "unable to find an available port")
		}
		max--
		if max == 0 {
			return nil, 0, errors.Wrapf(err, "unable to find an available port (from %d to %d)", preferedPort, port)
		}
		port++
	}
	if preferedPort == 0 {
		port = ln.Addr().(*net.TCPAddr).Port
	}
	return ln, port, nil
}
