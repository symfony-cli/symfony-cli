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

package inotify

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/syncthing/notify"
)

func Watch(path string, c chan<- notify.EventInfo, events ...notify.Event) error {
	if filepath.Base(path) == "..." {
		// This means a recursive watch
		return simpleWatch(path, c, events...)
	} else if fi, err := os.Stat(path); err != nil {
		return errors.WithStack(err)
	} else if fi.IsDir() {
		return simpleWatch(path, c, events...)
	}

	wrappedCh := make(chan notify.EventInfo, 10)
	if err := notify.Watch(filepath.Dir(path), wrappedCh, events...); err != nil {
		return errors.WithStack(err)
	}

	go func() {
		for {
			e := <-wrappedCh
			if e == nil {
				continue
			}
			if path != e.Path() {
				continue
			}
			c <- e
		}
	}()

	return nil
}
