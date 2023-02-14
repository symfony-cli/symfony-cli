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
	"github.com/pkg/errors"
	"github.com/syncthing/notify"
)

// Create, Remove, Write and Rename are the only event values guaranteed to be
// present on all platforms.
const (
	Create = notify.Create
	Remove = notify.Remove
	Write  = notify.Write
	Rename = notify.Rename

	// All is handful alias for all platform-independent event values.
	All = Create | Remove | Write | Rename
)

type EventInfo = notify.EventInfo

func Stop(c chan<- EventInfo) {
	notify.Stop(c)
}

func simpleWatch(path string, c chan<- EventInfo, events ...notify.Event) error {
	if err := notify.Watch(path, c, events...); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
