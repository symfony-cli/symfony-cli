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

package envs

import (
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type DotEnvSuite struct{}

var _ = Suite(&DotEnvSuite{})

func (s *DotEnvSuite) TestLoadDotEnvKeepsExportedVariables(c *C) {
	dir := c.MkDir()
	c.Assert(os.WriteFile(filepath.Join(dir, ".env"), []byte("EXPORTED=dotenv\nEXPORTED_EMPTY=dotenv\nDOTENV_ONLY=dotenv\n"), 0644), IsNil)

	for k, v := range map[string]string{"EXPORTED": "exported", "EXPORTED_EMPTY": ""} {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}
	for _, k := range []string{"APP_ENV", "SYMFONY_DOTENV_VARS"} {
		if v, ok := os.LookupEnv(k); ok {
			os.Unsetenv(k)
			defer os.Setenv(k, v)
		}
	}

	vars := LoadDotEnv(map[string]string{}, dir)

	c.Check(vars["DOTENV_ONLY"], Equals, "dotenv")
	c.Check(vars["SYMFONY_DOTENV_VARS"], Equals, "DOTENV_ONLY")
	_, ok := vars["EXPORTED"]
	c.Check(ok, Equals, false)
	_, ok = vars["EXPORTED_EMPTY"]
	c.Check(ok, Equals, false)
}
