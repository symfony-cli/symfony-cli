//go:build !windows

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

	. "gopkg.in/check.v1"
)

func (s *ExecutorSuite) TestSymlinkIsIdempotentWhenTargetIsASymlink(c *C) {
	dir := c.MkDir()
	target := filepath.Join(dir, "php-config8.4")
	c.Assert(os.WriteFile(target, []byte("#!/bin/sh\n"), 0755), IsNil)
	alternative := filepath.Join(dir, "php-config")
	c.Assert(os.Symlink(target, alternative), IsNil)

	link := filepath.Join(c.MkDir(), "php-config")
	c.Assert(symlink(alternative, link), IsNil)
	c.Assert(symlink(alternative, link), IsNil)
}

func (s *ExecutorSuite) TestSymlinkFailsWhenLinkPointsElsewhere(c *C) {
	dir := c.MkDir()
	link := filepath.Join(dir, "php-config")
	c.Assert(os.Symlink(filepath.Join(dir, "other"), link), IsNil)

	c.Assert(symlink(filepath.Join(dir, "php-config8.4"), link), NotNil)
}
