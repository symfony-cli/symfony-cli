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
	"path/filepath"

	. "gopkg.in/check.v1"
)

type UtilsSuite struct{}

var _ = Suite(&UtilsSuite{})

func (s *UtilsSuite) TestIsPHPScript(c *C) {
	dir, err := filepath.Abs("testdata/php_scripts")
	c.Assert(err, IsNil)

	c.Assert(isPHPScript(""), Equals, false)
	c.Assert(isPHPScript(filepath.Join(dir, "unknown")), Equals, false)
	c.Assert(isPHPScript(filepath.Join(dir, "invalid")), Equals, false)

	for _, validScripts := range []string{
		"usual-one",
		"debian-style",
		"custom-one",
		"plain-one.php",
	} {
		c.Assert(isPHPScript(filepath.Join(dir, validScripts)), Equals, true)
	}
}

func (s *UtilsSuite) TestIsPHPScriptNixWrapper(c *C) {
	dir, err := filepath.Abs("testdata/php_scripts")
	c.Assert(err, IsNil)

	// Test Nix wrapper with valid PHP script
	c.Assert(isPHPScript(filepath.Join(dir, "nix-wrapper")), Equals, true,
		Commentf("Nix wrapper with valid PHP wrapped file should be detected as PHP script"))

	// Test Nix wrapper with invalid wrapped file
	c.Assert(isPHPScript(filepath.Join(dir, "nix-wrapper-invalid")), Equals, false,
		Commentf("Nix wrapper with invalid wrapped file should not be detected as PHP script"))
}
