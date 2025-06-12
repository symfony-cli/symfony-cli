//go:build !windows
// +build !windows

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

package php

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

func shouldSignalBeIgnored(sig os.Signal) bool {
	// this one in particular should be skipped as we don't want to
	// send it back to child because it's about it
	return sig == syscall.SIGCHLD
}

func symlink(oldname, newname string) error {
	err := errors.WithStack(os.Symlink(oldname, newname))

	if os.IsExist(errors.Cause(err)) {
		if target, _ := filepath.EvalSymlinks(newname); target == oldname {
			return nil
		}
	}

	return err
}
